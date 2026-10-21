"""Check mounted secret wiring and MongoDB endpoint validation after rendering."""

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
    def test_mounted_secrets(self):
        result = render("-f", str(CHART / "ci/secret-files-values.yaml"))
        self.assertEqual(result.returncode, 0, result.stderr)
        docs = [doc for doc in yaml.safe_load_all(result.stdout) if doc]
        deployment = next(doc for doc in docs if doc["kind"] == "Deployment"
                          and doc["metadata"]["name"].endswith("-apiserver"))
        pod = deployment["spec"]["template"]["spec"]
        container = pod["containers"][0]
        files = [env for env in container["env"] if env["name"].endswith("_FILE")]
        self.assertEqual(len(files), 8)
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
