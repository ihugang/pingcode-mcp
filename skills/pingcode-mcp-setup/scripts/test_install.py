"""Offline unit tests for the installer. PRD FR-003/FR-010/FR-011; OpenSpec task 2.3."""

from __future__ import annotations

import contextlib
import hashlib
import io
import json
import tarfile
import tempfile
import unittest
import zipfile
from unittest import mock
from pathlib import Path

import install


class PlatformTests(unittest.TestCase):
    def test_platform_aliases(self) -> None:
        self.assertEqual(install.target_platform("Darwin", "arm64"), ("darwin", "arm64"))
        self.assertEqual(install.target_platform("Linux", "x86_64"), ("linux", "amd64"))
        self.assertEqual(install.target_platform("Windows", "AMD64"), ("windows", "amd64"))

    def test_unsupported_platform_is_rejected(self) -> None:
        with self.assertRaises(install.InstallError):
            install.target_platform("Plan9", "amd64")
        with self.assertRaises(install.InstallError):
            install.target_platform("Linux", "mips")

    def test_tag_and_release_names(self) -> None:
        self.assertEqual(install.release_asset_name("v1.2.3", "darwin", "arm64"), "pingcode-mcp_1.2.3_darwin_arm64.tar.gz")
        self.assertEqual(install.release_asset_name("v1.2.3", "windows", "amd64"), "pingcode-mcp_1.2.3_windows_amd64.zip")
        with self.assertRaises(install.InstallError):
            install.validate_tag("main")


class ReleaseValidationTests(unittest.TestCase):
    def test_download_opener_uses_environment_proxy_settings(self) -> None:
        url = "https://github.com/ihugang/pingcode-mcp/releases/download/v1.2.3/asset.tar.gz"
        proxies = {
            "http": "http://proxy.example:8080",
            "https": "http://proxy.example:8080",
            "no": "localhost,127.0.0.1",
        }
        response = mock.MagicMock()
        response.__enter__.return_value = response
        response.geturl.return_value = url
        response.read.return_value = b"release asset"
        opener = mock.MagicMock()
        opener.open.return_value = response

        with mock.patch.object(install.urllib.request, "getproxies", return_value=proxies) as getproxies, \
            mock.patch.object(install.urllib.request, "build_opener", return_value=opener) as build_opener:
            result = install.request_bytes(url, timeout=17)

        self.assertEqual(result, b"release asset")
        getproxies.assert_called_once_with()
        proxy_handler = build_opener.call_args.args[0]
        self.assertIsInstance(proxy_handler, install.urllib.request.ProxyHandler)
        self.assertEqual(proxy_handler.proxies, proxies)
        opener.open.assert_called_once_with(mock.ANY, timeout=17)

    def test_all_proxy_is_used_when_protocol_proxy_is_missing(self) -> None:
        url = "https://github.com/ihugang/pingcode-mcp/releases/latest"
        proxies = {"all": "http://proxy.example:8080", "no": "localhost"}
        with mock.patch.object(install.urllib.request, "getproxies", return_value=proxies):
            handler = install.environment_proxy_handler(url)
        self.assertEqual(handler.proxies["https"], "http://proxy.example:8080")

    def test_unsupported_all_proxy_fails_without_exposing_value_or_bypassing(self) -> None:
        marker = "redacted-proxy-marker"
        proxies = {"all": f"socks5h://user:{marker}@proxy.example:1080"}
        with mock.patch.object(install.urllib.request, "getproxies", return_value=proxies), \
            mock.patch.object(install.urllib.request, "proxy_bypass", return_value=False), \
            mock.patch.object(install.urllib.request, "build_opener") as build_opener:
            with self.assertRaises(install.InstallError) as caught:
                install.request_bytes("https://github.com/ihugang/pingcode-mcp/releases/latest")
        self.assertNotIn(marker, str(caught.exception))
        build_opener.assert_not_called()

    def test_no_proxy_bypasses_unsupported_all_proxy(self) -> None:
        proxies = {"all": "socks5h://proxy.example:1080", "no": "github.com"}
        with mock.patch.object(install.urllib.request, "getproxies", return_value=proxies), \
            mock.patch.object(install.urllib.request, "proxy_bypass", return_value=True):
            handler = install.environment_proxy_handler("https://github.com/ihugang/pingcode-mcp/releases/latest")
        self.assertIsInstance(handler, install.urllib.request.ProxyHandler)

    def test_invalid_proxy_error_does_not_expose_proxy_value(self) -> None:
        secret_proxy = "http://user:redacted-proxy-marker@proxy.example:invalid"
        with mock.patch.object(
            install.urllib.request, "ProxyHandler", side_effect=ValueError(f"invalid proxy {secret_proxy}")
        ):
            with self.assertRaises(install.InstallError) as caught:
                install.request_bytes("https://github.com/ihugang/pingcode-mcp/releases/latest")
        self.assertNotIn("redacted-proxy-marker", str(caught.exception))

    def test_checksum_accepts_expected_asset(self) -> None:
        payload = b"fake archive"
        name = "pingcode-mcp_1.2.3_linux_amd64.tar.gz"
        sums = f"{hashlib.sha256(payload).hexdigest()}  {name}\n".encode()
        install.verify_checksum(payload, sums, name)

    def test_checksum_rejects_mismatch_and_duplicate(self) -> None:
        payload = b"fake archive"
        name = "pingcode-mcp_1.2.3_linux_amd64.tar.gz"
        bad = f"{'0' * 64}  {name}\n".encode()
        with self.assertRaises(install.InstallError):
            install.verify_checksum(payload, bad, name)
        digest = hashlib.sha256(payload).hexdigest()
        duplicate = f"{digest}  {name}\n{digest}  {name}\n".encode()
        with self.assertRaises(install.InstallError):
            install.parse_checksum(duplicate, name)

    def test_tar_archive_extracts_only_expected_binary(self) -> None:
        archive = io.BytesIO()
        with tarfile.open(fileobj=archive, mode="w:gz") as bundle:
            info = tarfile.TarInfo("pingcode-mcp")
            info.size = 7
            bundle.addfile(info, io.BytesIO(b"program"))
        result = install.unpack_expected_binary(
            archive.getvalue(), "pingcode-mcp_1.2.3_linux_amd64.tar.gz", "pingcode-mcp"
        )
        self.assertEqual(result, b"program")

    def test_archive_member_validation_rejects_normalized_paths(self) -> None:
        self.assertFalse(install.safe_archive_member("folder/./file"))
        self.assertFalse(install.safe_archive_member("folder//file"))
        self.assertTrue(install.safe_archive_member("folder/file"))

    def test_release_binary_size_is_checked_before_read(self) -> None:
        archive = io.BytesIO()
        with zipfile.ZipFile(archive, "w") as bundle:
            bundle.writestr("pingcode-mcp.exe", b"program")
        with mock.patch.object(install, "MAX_DOWNLOAD_BYTES", 3):
            with self.assertRaisesRegex(install.InstallError, "100 MiB"):
                install.unpack_expected_binary(archive.getvalue(), "asset.zip", "pingcode-mcp.exe")

    def test_source_archive_has_total_unpacked_size_limit(self) -> None:
        archive = io.BytesIO()
        with tarfile.open(fileobj=archive, mode="w:gz") as bundle:
            for name, content in (("repo/a", b"123"), ("repo/b", b"456")):
                info = tarfile.TarInfo(name)
                info.size = len(content)
                bundle.addfile(info, io.BytesIO(content))
        with tempfile.TemporaryDirectory() as temp_name, mock.patch.object(install, "MAX_SOURCE_BYTES", 5):
            with self.assertRaisesRegex(install.InstallError, "500 MiB"):
                install.extract_source(archive.getvalue(), Path(temp_name) / "source")

    def test_zip_archive_rejects_traversal_and_extra_files(self) -> None:
        traversal = io.BytesIO()
        with zipfile.ZipFile(traversal, "w") as bundle:
            bundle.writestr("../outside", b"no")
        with self.assertRaises(install.InstallError):
            install.unpack_expected_binary(traversal.getvalue(), "asset.zip", "pingcode-mcp.exe")

        extra = io.BytesIO()
        with zipfile.ZipFile(extra, "w") as bundle:
            bundle.writestr("pingcode-mcp.exe", b"program")
            bundle.writestr("README.txt", b"unexpected")
        with self.assertRaises(install.InstallError):
            install.unpack_expected_binary(extra.getvalue(), "asset.zip", "pingcode-mcp.exe")


class ConfigurationTests(unittest.TestCase):
    def test_json_merge_preserves_unrelated_servers(self) -> None:
        with tempfile.TemporaryDirectory() as temp_name:
            path = Path(temp_name) / "mcp.json"
            path.write_text(json.dumps({"mcpServers": {"other": {"command": "keep"}}}), encoding="utf-8")
            state, document = install.json_config_state(path, "mcpServers", "kiro", Path("/bin/pingcode-mcp"))
            self.assertEqual(state, "new")
            result = json.loads(install.json_config_bytes(document, "mcpServers", "kiro", Path("/bin/pingcode-mcp")))
            self.assertEqual(result["mcpServers"]["other"], {"command": "keep"})
            self.assertEqual(result["mcpServers"]["pingcode"]["env"], {"PINGCODE_READ_ONLY": "true"})
            self.assertNotIn("autoApprove", result["mcpServers"]["pingcode"])

    def test_cursor_configuration_declares_stdio_transport(self) -> None:
        config = install.server_config("cursor", Path("/bin/pingcode-mcp"))
        self.assertEqual(config["type"], "stdio")
        self.assertEqual(config["env"], {"PINGCODE_READ_ONLY": "true"})

    def test_json_conflict_is_detected_without_showing_values(self) -> None:
        with tempfile.TemporaryDirectory() as temp_name:
            path = Path(temp_name) / "mcp.json"
            path.write_text(json.dumps({"mcpServers": {"pingcode": {"env": {"TOKEN": "private"}}}}), encoding="utf-8")
            state, _ = install.json_config_state(path, "mcpServers", "cursor", Path("/bin/pingcode-mcp"))
            self.assertEqual(state, "conflict")
            self.assertNotIn("private", repr(state))

    def test_toml_append_and_replace_preserve_sibling_tables(self) -> None:
        original = '[mcp_servers.other]\ncommand = "other"\n'
        updated = install.updated_toml(original, Path("/bin/pingcode-mcp"), False).decode()
        parsed = install.tomllib.loads(updated)
        self.assertEqual(parsed["mcp_servers"]["other"]["command"], "other")
        self.assertEqual(parsed["mcp_servers"]["pingcode"]["env"]["PINGCODE_READ_ONLY"], "true")

        conflict = '[mcp_servers.pingcode]\ncommand = "old"\n\n[mcp_servers.other]\ncommand = "other"\n'
        replaced = install.updated_toml(conflict, Path("/new/pingcode-mcp"), True).decode()
        parsed_replaced = install.tomllib.loads(replaced)
        self.assertEqual(parsed_replaced["mcp_servers"]["pingcode"]["command"], "/new/pingcode-mcp")
        self.assertEqual(parsed_replaced["mcp_servers"]["other"]["command"], "other")

    def test_toml_replace_requires_explicit_flag(self) -> None:
        conflict = '[mcp_servers.pingcode]\ncommand = "old"\n'
        with self.assertRaises(install.InstallError):
            install.updated_toml(conflict, Path("/new/pingcode-mcp"), False)

    def test_atomic_write_preserves_existing_permissions(self) -> None:
        with tempfile.TemporaryDirectory() as temp_name:
            path = Path(temp_name) / "mcp.json"
            path.write_text("{}\n", encoding="utf-8")
            path.chmod(0o640)
            install.write_atomic(path, b'{"ok":true}\n')
            self.assertEqual(path.read_bytes(), b'{"ok":true}\n')
            self.assertEqual(path.stat().st_mode & 0o777, 0o640)

    def test_json_config_refuses_symlink_target(self) -> None:
        with tempfile.TemporaryDirectory() as temp_name:
            root = Path(temp_name)
            actual = root / "actual.json"
            actual.write_text("{}", encoding="utf-8")
            link = root / "mcp.json"
            try:
                link.symlink_to(actual)
            except (OSError, NotImplementedError):
                self.skipTest("symlinks are unavailable")
            with self.assertRaises(install.InstallError):
                install.load_json_config(link)

    def test_workspace_paths_are_client_specific(self) -> None:
        with tempfile.TemporaryDirectory() as temp_name:
            workspace = Path(temp_name)
            self.assertEqual(
                install.json_client_path("kiro", "workspace", workspace, Path("/home/user"))[0],
                workspace / ".kiro" / "settings" / "mcp.json",
            )
            self.assertEqual(
                install.json_client_path("vscode", "workspace", workspace, Path("/home/user"))[0],
                workspace / ".mcp.json",
            )
            with self.assertRaises(install.InstallError):
                install.json_client_path("cursor", "workspace", None, Path("/home/user"))


class InstallerControlTests(unittest.TestCase):
    def test_dry_run_does_not_fetch_or_write(self) -> None:
        with tempfile.TemporaryDirectory() as temp_name:
            home = Path(temp_name)
            args = install.parse_args(["--client", "kiro", "--scope", "user", "--dry-run"])
            with mock.patch.object(Path, "home", return_value=home), mock.patch.object(
                install, "target_platform", return_value=("linux", "amd64")
            ), mock.patch.object(install, "request_json", side_effect=AssertionError("network used")), contextlib.redirect_stdout(
                io.StringIO()
            ):
                self.assertEqual(install.run_install(args), 0)
            self.assertFalse((home / ".local" / "bin").exists())
            self.assertFalse((home / ".kiro").exists())

    def test_mocked_release_install_writes_binary_and_client_config(self) -> None:
        with tempfile.TemporaryDirectory() as temp_name:
            home = Path(temp_name)
            archive = b"mock release archive"
            checksum = hashlib.sha256(archive).hexdigest()
            sums = f"{checksum}  pingcode-mcp_1.2.3_linux_amd64.tar.gz\n".encode()
            release = {"tag_name": "v1.2.3", "draft": False, "prerelease": False}
            args = install.parse_args(["--client", "kiro", "--scope", "user", "--yes"])
            with mock.patch.object(Path, "home", return_value=home), \
                mock.patch.object(install, "target_platform", return_value=("linux", "amd64")), \
                mock.patch.object(install, "get_release", return_value=release), \
                mock.patch.object(install, "asset_url", side_effect=lambda _release, name: f"https://github.com/{name}"), \
                mock.patch.object(install, "request_bytes", side_effect=[archive, sums]), \
                mock.patch.object(install, "unpack_expected_binary", return_value=b"mock executable"), \
                mock.patch.object(install, "verify_binary_version"), \
                mock.patch.object(install, "verify_mcp"), \
                mock.patch.object(install, "verify_attestation", return_value="attestation mocked"), \
                contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(install.run_install(args), 0)
            binary = home / ".local" / "bin" / "pingcode-mcp"
            config = home / ".kiro" / "settings" / "mcp.json"
            self.assertEqual(binary.read_bytes(), b"mock executable")
            document = json.loads(config.read_text(encoding="utf-8"))
            self.assertEqual(document["mcpServers"]["pingcode"]["env"], {"PINGCODE_READ_ONLY": "true"})

    def test_bad_release_checksum_leaves_files_unchanged(self) -> None:
        with tempfile.TemporaryDirectory() as temp_name:
            home = Path(temp_name)
            archive = b"bad archive"
            sums = f"{'0' * 64}  pingcode-mcp_1.2.3_linux_amd64.tar.gz\n".encode()
            release = {"tag_name": "v1.2.3", "draft": False, "prerelease": False}
            args = install.parse_args(["--client", "kiro", "--scope", "user", "--yes"])
            with mock.patch.object(Path, "home", return_value=home), \
                mock.patch.object(install, "target_platform", return_value=("linux", "amd64")), \
                mock.patch.object(install, "get_release", return_value=release), \
                mock.patch.object(install, "asset_url", side_effect=lambda _release, name: f"https://github.com/{name}"), \
                mock.patch.object(install, "request_bytes", side_effect=[archive, sums]), \
                contextlib.redirect_stdout(io.StringIO()):
                with self.assertRaisesRegex(install.InstallError, "SHA-256"):
                    install.run_install(args)
            self.assertFalse((home / ".local" / "bin" / "pingcode-mcp").exists())
            self.assertFalse((home / ".kiro" / "settings" / "mcp.json").exists())

    def test_user_cancel_stops_before_mutation(self) -> None:
        args = install.parse_args(["--client", "kiro"])
        with mock.patch.object(install.sys, "stdin") as fake_stdin, mock.patch("builtins.input", return_value="n"):
            fake_stdin.isatty.return_value = True
            with self.assertRaises(install.InstallError):
                install.prompt_to_continue(args)

    def test_mcp_timeout_is_reported_without_details(self) -> None:
        with tempfile.TemporaryDirectory() as temp_name:
            executable = Path(temp_name) / "silent-mcp"
            executable.write_text("#!/usr/bin/env python3\nimport time\ntime.sleep(3)\n", encoding="utf-8")
            executable.chmod(0o755)
            with self.assertRaisesRegex(install.InstallError, "初始化超时"):
                install.verify_mcp(executable, timeout=1)


if __name__ == "__main__":
    unittest.main()
