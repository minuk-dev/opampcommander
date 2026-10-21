"""Render-contract checks. Run with Python 3 + PyYAML after helm dependency build."""

import pathlib
import subprocess
import unittest

import yaml

CHART = pathlib.Path(__file__).resolve().parents[1]
PREVIEW = CHART / "ci/grafana-values.yaml"


def render(*args):
    return subprocess.run(
        ["helm", "template", "preview", str(CHART), *args],
        text=True, capture_output=True, check=False,
    )


class GrafanaChartTest(unittest.TestCase):
    def documents(self, *args):
        result = render(*args)
        self.assertEqual(result.returncode, 0, result.stderr)
        return [doc for doc in yaml.safe_load_all(result.stdout) if doc]

    def test_default_has_no_grafana_resources(self):
        docs = self.documents("--kube-version", "1.23.0")
        self.assertTrue(docs)
        self.assertFalse(any("grafana" in doc["metadata"]["name"] for doc in docs))
        self.assertFalse(any("apps.yaml" in doc.get("data", {}) for doc in docs))

    def test_enabled_requires_explicit_plugin_image(self):
        for extra in ([], ["--set", "grafana.image.repository=custom"]):
            result = render("--set", "grafana.enabled=true", *extra)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("requires grafana.image.repository and grafana.image.tag", result.stderr)

    def test_preview_provisioning_and_persistence(self):
        docs = self.documents(
            "-f", str(PREVIEW), "--set", "grafana.fullnameOverride=custom-grafana",
            "--set", "grafana.persistence.enabled=true",
            "--set", "grafana.admin.existingSecret=grafana-admin",
        )
        deploy = next(d for d in docs if d["kind"] == "Deployment" and d["metadata"]["name"] == "custom-grafana")
        pod = deploy["spec"]["template"]["spec"]
        container = next(c for c in pod["containers"] if c["name"] == "grafana")
        self.assertEqual(container["image"], "docker.io/opampcommander-grafana:dev")
        self.assertFalse(pod["automountServiceAccountToken"])
        self.assertFalse(any(d["kind"] in ("ClusterRole", "ClusterRoleBinding") for d in docs))
        env = {e["name"]: e for e in container["env"]}
        self.assertEqual(env["GF_PATHS_PLUGINS"]["value"], "/opt/opampcommander/plugins")
        self.assertEqual(env["GF_SECURITY_ADMIN_PASSWORD"]["valueFrom"]["secretKeyRef"]["name"], "grafana-admin")
        self.assertFalse(any(d["kind"] == "Secret" and d["metadata"]["name"] == "custom-grafana" for d in docs))
        self.assertTrue(any("persistentVolumeClaim" in v for v in pod["volumes"]))
        self.assertFalse(any(m["mountPath"].startswith("/opt/opampcommander") for m in container["volumeMounts"]))
        mount = next(v for v in pod["volumes"] if v["name"] == "opampcommander-app")
        config = next(d for d in docs if d["kind"] == "ConfigMap" and d["metadata"]["name"] == mount["configMap"]["name"])
        self.assertEqual(yaml.safe_load(config["data"]["apps.yaml"])["apps"][0]["type"], "minukdev-opampcommander-app")
        grafana = next(d for d in docs if d["kind"] == "ConfigMap" and d["metadata"]["name"] == "custom-grafana")
        self.assertIn("[auth.anonymous]\nenabled = false", grafana["data"]["grafana.ini"])
        source = yaml.safe_load(grafana["data"]["opampcommander.yaml"])["datasources"][0]
        self.assertEqual(source["type"], "minukdev-opampcommander-datasource")
        self.assertFalse(source["editable"])
        self.assertNotIn("secureJsonData", source)


if __name__ == "__main__":
    unittest.main()
