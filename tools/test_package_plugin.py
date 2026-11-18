"""Check that local build leftovers cannot enter a distributable plugin."""
from __future__ import annotations

import importlib.util
import json
import shutil
import tempfile
import unittest
import zipfile
from pathlib import Path

SPEC = importlib.util.spec_from_file_location("package_plugin", Path(__file__).with_name("package-plugin.py"))
assert SPEC and SPEC.loader
PACKAGER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(PACKAGER)
REPO = Path(__file__).resolve().parent.parent


class PackageTests(unittest.TestCase):
    """Use only disposable packages and a synthetic executable fixture."""

    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory(prefix="seshy-package-test-")
        self.addCleanup(self.temporary.cleanup)
        self.output = Path(self.temporary.name)
        self.root = self.output / "seshy"
        shutil.copytree(REPO / "plugins/seshy", self.root)
        shutil.copy(REPO / "LICENSE", self.root / "LICENSE")
        (self.root / "bin").mkdir()
        self.runtime = self.root / "bin/seshy-darwin-arm64"
        self.runtime.write_text("synthetic runtime fixture\n")
        self.runtime.chmod(0o755)

    def test_archive_contains_only_reviewed_files_and_executable_mode(self) -> None:
        archive = PACKAGER.package(self.output)
        with zipfile.ZipFile(archive) as result:
            self.assertEqual(len(result.namelist()), 8)
            self.assertTrue(all(name.startswith("seshy/") for name in result.namelist()))
            self.assertTrue(result.getinfo("seshy/bin/seshy-darwin-arm64").external_attr >> 16 & 0o111)

    def test_rejects_private_leftovers_in_each_package_directory(self) -> None:
        for directory in ("assets", "skills/recover-session", "bin"):
            with self.subTest(directory=directory):
                extra = self.root / directory / "history.jsonl"
                extra.write_text("synthetic private leftover\n")
                with self.assertRaisesRegex(ValueError, "Unexpected package files"):
                    PACKAGER.package(self.output)
                extra.unlink()

    def test_rejects_symlinks_before_reading_manifest(self) -> None:
        manifest = self.root / "plugin.json"
        manifest.unlink()
        manifest.symlink_to(self.output / "absent.json")
        with self.assertRaisesRegex(ValueError, "Symlinks cannot be packaged"):
            PACKAGER.package(self.output)

    def test_rejects_missing_or_nonexecutable_runtime(self) -> None:
        self.runtime.chmod(0o644)
        with self.assertRaisesRegex(ValueError, "not executable"):
            PACKAGER.package(self.output)
        self.runtime.unlink()
        with self.assertRaisesRegex(ValueError, "Build the runtime"):
            PACKAGER.package(self.output)

    def test_rejects_archive_path_in_version(self) -> None:
        manifest = self.root / "plugin.json"
        data = json.loads(manifest.read_text())
        data["version"] = "../../unreviewed"
        manifest.write_text(json.dumps(data))
        with self.assertRaisesRegex(ValueError, "portable semantic version"):
            PACKAGER.package(self.output)


if __name__ == "__main__":
    unittest.main()
