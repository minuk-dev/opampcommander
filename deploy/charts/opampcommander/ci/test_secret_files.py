"""Check mounted Secret wiring, runtime permissions and endpoint validation."""

import pathlib
import subprocess
import unittest

import yaml

CHART = pathlib.Path(__file__).resolve().parents[1]


def render(*args):
    return subprocess.run(
        ["helm", "template", "preview", str(CHART), *args],
        text=True, capture_output=True, check=False,
    )


class SecretFilesChartTest(unittest.TestCase):
    def test_ingress_exposes_only_opamp(self):
        result = render("--set", "apiserver.ingress.enabled=true",
                        "--set", "apiserver.ingress.hosts[0].host=agents.example.com",
                        "--set", "apiserver.ingress.hosts[0].paths[0].path=/")
        self.assertEqual(result.returncode, 0, result.stderr)
        docs = [doc for doc in yaml.safe_load_all(result.stdout) if doc]
        ingress = next(doc for doc in docs if doc["kind"] == "Ingress")
        paths = ingress["spec"]["rules"][0]["http"]["paths"]
        self.assertEqual([(path["path"], path["pathType"]) for path in paths],
                         [("/api/v1/opamp", "Exact")])

    def test_runtime_permissions(self):
        result = render("--namespace", "gateway",
                        "-f", str(CHART / "ci/runtime-identity-values.yaml"))
        self.assertEqual(result.returncode, 0, result.stderr)
        docs = {doc["kind"]: doc for doc in yaml.safe_load_all(result.stdout) if doc}
        expected_subjects = [{"kind": "ServiceAccount",
                              "name": docs["Deployment"]["spec"]["template"]["spec"]["serviceAccountName"],
                              "namespace": "gateway"}]
        self.assertEqual(docs["ClusterRole"]["rules"], [
            {"apiGroups": [""], "resources": ["namespaces"], "verbs": ["get"]}])
        self.assertEqual(docs["Role"]["rules"], [
            {"apiGroups": [""], "resources": ["configmaps"], "verbs": ["get"],
             "resourceNames": ["opampcommander-operator-opamp-installation"]}])
        for kind in ("Role", "RoleBinding"):
            self.assertEqual(docs[kind]["metadata"]["namespace"], "opampcommander-operator-system")
        for kind in ("Role", "ClusterRole"):
            binding = docs[kind + "Binding"]
            self.assertEqual(binding["subjects"], expected_subjects)
            self.assertEqual(binding["roleRef"]["name"], docs[kind]["metadata"]["name"])
        self.assertTrue(docs["Deployment"]["spec"]["template"]["spec"]["automountServiceAccountToken"])

    def test_runtime_requires_a_complete_identity_and_dedicated_token(self):
        partial = render("--set", "apiserver.config.runtimeAPI.installationUID=uid")
        self.assertNotEqual(partial.returncode, 0)
        self.assertIn("requires installationNamespace, installationName and installationUID", partial.stderr)
        for value in ("apiserver.serviceAccount.create=false",
                      "apiserver.serviceAccount.automountServiceAccountToken=false"):
            with self.subTest(value=value):
                result = render("-f", str(CHART / "ci/runtime-identity-values.yaml"), "--set", value)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("requires a dedicated apiserver ServiceAccount", result.stderr)

    def test_mounted_secrets(self):
        result = render("-f", str(CHART / "ci/secret-files-values.yaml"))
        self.assertEqual(result.returncode, 0, result.stderr)
        docs = [doc for doc in yaml.safe_load_all(result.stdout) if doc]
        deployment = next(doc for doc in docs if doc["kind"] == "Deployment"
                          and doc["metadata"]["name"].endswith("-apiserver"))
        pod = deployment["spec"]["template"]["spec"]
        container = pod["containers"][0]
        files = [env for env in container["env"] if env["name"].endswith("_FILE")]
        self.assertEqual(len(files), 3)
        self.assertTrue(all(env["value"].startswith("/run/secrets/opampcommander/")
                            for env in files))
        self.assertFalse(any("valueFrom" in env and "secretKeyRef" in env["valueFrom"]
                             for env in container["env"]))
        mount = next(m for m in container["volumeMounts"] if m["name"] == "secret-files")
        self.assertTrue(mount["readOnly"])
        self.assertEqual(mount["mountPath"], "/run/secrets/opampcommander")
        volume = next(v for v in pod["volumes"] if v["name"] == "secret-files")
        self.assertEqual(volume["secret"]["secretName"], "opampcommander-secret-files")
        self.assertFalse(any(doc["kind"] == "Secret" for doc in docs))

    def test_mongodb_requires_an_endpoint_source(self):
        base = ["--set", "apiserver.config.database.type=mongodb",
                "--set-json", "apiserver.config.database.endpoints=[]"]
        for name, value, accepted in [
            ("DATABASE_ENDPOINTS_FILE", "/run/secrets/uri", True),
            ("DATABASE_ENDPOINTS", "mongodb://mongo:27017", True),
            ("DATABASE_ENDPOINTS_FILE", "", False),
            ("UNRELATED", "value", False),
        ]:
            with self.subTest(name=name, value=value):
                result = render(*base, "--set", f"apiserver.extraEnv[0].name={name}",
                                "--set-string", f"apiserver.extraEnv[0].value={value}")
                self.assertEqual(result.returncode == 0, accepted, result.stderr)
        result = render(*base, "--set", "apiserver.extraEnv[0].name=DATABASE_ENDPOINTS",
                        "--set", "apiserver.extraEnv[0].valueFrom.secretKeyRef.name=db",
                        "--set", "apiserver.extraEnv[0].valueFrom.secretKeyRef.key=uri")
        self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == "__main__":
    unittest.main()
