#!/usr/bin/env python3
"""Install the PingCode MCP CLI and register it with one local client.

Traceability: PRD FR-003/FR-010, OpenSpec change
pingcode-mcp-setup-skill tasks 2.1-2.3. PingCode requirement ID: PENDING.
"""

from __future__ import annotations

import argparse
import hashlib
import io
import json
import os
import platform
import re
import shutil
import stat
import subprocess
import sys
import tarfile
import tempfile
import threading
import time
import urllib.error
import urllib.parse
import urllib.request
import zipfile
from pathlib import Path, PurePosixPath
from typing import Any

try:
    import tomllib
except ImportError:  # Python 3.10 and earlier receive a friendly CLI error below.
    tomllib = None  # type: ignore[assignment]


OWNER = "ihugang"
REPOSITORY = "pingcode-mcp"
REPOSITORY_SLUG = f"{OWNER}/{REPOSITORY}"
API_ROOT = f"https://api.github.com/repos/{REPOSITORY_SLUG}"
USER_AGENT = "pingcode-mcp-setup/1"
MAX_DOWNLOAD_BYTES = 100 * 1024 * 1024
MAX_SOURCE_BYTES = 500 * 1024 * 1024
EXPECTED_TOOLS = {
    "pingcode_list_projects",
    "pingcode_get_project",
    "pingcode_list_work_items",
    "pingcode_get_work_item",
    "pingcode_create_work_item",
    "pingcode_update_work_item",
}


class InstallError(Exception):
    """An expected, safe-to-display installer error."""


def target_platform(system: str | None = None, machine: str | None = None) -> tuple[str, str]:
    """Map Python platform names to Go release targets."""
    system = (system or platform.system()).lower()
    machine = (machine or platform.machine()).lower()
    systems = {"darwin": "darwin", "linux": "linux", "windows": "windows"}
    architectures = {
        "amd64": "amd64",
        "x86_64": "amd64",
        "x64": "amd64",
        "arm64": "arm64",
        "aarch64": "arm64",
    }
    if system not in systems:
        raise InstallError(f"暂不支持操作系统：{system}。支持 macOS、Linux 和 Windows。")
    if machine not in architectures:
        raise InstallError(f"暂不支持处理器架构：{machine}。支持 amd64 和 arm64。")
    return systems[system], architectures[machine]


def release_asset_name(tag: str, goos: str, goarch: str) -> str:
    version = tag.removeprefix("v")
    extension = "zip" if goos == "windows" else "tar.gz"
    return f"pingcode-mcp_{version}_{goos}_{goarch}.{extension}"


def binary_name(goos: str) -> str:
    return "pingcode-mcp.exe" if goos == "windows" else "pingcode-mcp"


def validate_tag(tag: str) -> str:
    if not re.fullmatch(r"v\d+\.\d+\.\d+", tag):
        raise InstallError("版本必须是稳定版 tag，格式例如 v1.2.3。")
    return tag


def request_bytes(url: str, timeout: int = 30) -> bytes:
    request = urllib.request.Request(
        url,
        headers={"Accept": "application/vnd.github+json", "User-Agent": USER_AGENT},
    )
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            final_host = urllib.parse.urlparse(response.geturl()).hostname or ""
            if (
                final_host != "github.com"
                and not final_host.endswith(".github.com")
                and not final_host.endswith(".githubusercontent.com")
            ):
                raise InstallError("GitHub 下载跳转到了非 GitHub 域名，已停止。")
            data = response.read(MAX_DOWNLOAD_BYTES + 1)
    except InstallError:
        raise
    except (urllib.error.URLError, TimeoutError, OSError):
        raise InstallError("无法从 GitHub 获取 release 资产；请检查网络后重试。") from None
    if len(data) > MAX_DOWNLOAD_BYTES:
        raise InstallError("下载文件超过 100 MiB 安全上限。")
    return data


def request_json(url: str) -> dict[str, Any]:
    try:
        return json.loads(request_bytes(url).decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError):
        raise InstallError("GitHub 返回了无法识别的 release 信息。") from None


def asset_url(release: dict[str, Any], name: str) -> str:
    for asset in release.get("assets", []):
        if asset.get("name") == name:
            url = asset.get("browser_download_url", "")
            if url.startswith("https://github.com/"):
                return url
    raise InstallError(f"所选版本没有必需资产 {name}。")


def get_release(tag: str | None) -> dict[str, Any]:
    endpoint = f"{API_ROOT}/releases/latest" if tag is None else f"{API_ROOT}/releases/tags/{tag}"
    try:
        release = request_json(endpoint)
    except InstallError as exc:
        if tag is None:
            raise InstallError("仓库还没有可安装的正式 GitHub Release。请显式选择 --build-from-source，或稍后重试。") from exc
        raise InstallError(f"找不到正式版本 {tag}。") from exc
    if release.get("draft") or release.get("prerelease"):
        raise InstallError("选择的 GitHub Release 不是正式稳定版本。")
    actual_tag = release.get("tag_name", "")
    validate_tag(actual_tag)
    if tag is not None and actual_tag != tag:
        raise InstallError("GitHub 返回的版本与请求版本不一致。")
    return release


def parse_checksum(manifest: bytes, expected_name: str) -> str:
    try:
        lines = manifest.decode("ascii").splitlines()
    except UnicodeDecodeError:
        raise InstallError("SHA256SUMS 不是有效的 ASCII 文本。") from None
    matches: list[str] = []
    for line in lines:
        match = re.fullmatch(r"([0-9a-fA-F]{64})  (\*?)([^\s]+)", line.strip())
        if match and match.group(3) == expected_name:
            matches.append(match.group(1).lower())
    if len(matches) != 1:
        raise InstallError(f"SHA256SUMS 中缺少唯一的 {expected_name} 摘要。")
    return matches[0]


def verify_checksum(archive: bytes, manifest: bytes, expected_name: str) -> None:
    expected = parse_checksum(manifest, expected_name)
    actual = hashlib.sha256(archive).hexdigest()
    if actual != expected:
        raise InstallError("下载文件的 SHA-256 校验失败，未安装任何文件。")


def safe_archive_member(name: str) -> bool:
    path = PurePosixPath(name)
    return (
        bool(path.parts)
        and path.as_posix() == name.rstrip("/")
        and not path.is_absolute()
        and ".." not in path.parts
        and "\\" not in name
        and not re.match(r"^[A-Za-z]:", name)
        and "\x00" not in name
    )


def unpack_expected_binary(archive: bytes, archive_name: str, expected_binary: str) -> bytes:
    """Read only the expected regular file; never extract archive paths."""
    found: bytes | None = None
    try:
        if archive_name.endswith(".zip"):
            with zipfile.ZipFile(io.BytesIO(archive)) as bundle:
                for info in bundle.infolist():
                    if not safe_archive_member(info.filename):
                        raise InstallError("release 压缩包包含不安全路径，已拒绝安装。")
                    if info.is_dir():
                        continue
                    file_type = (info.external_attr >> 16) & 0o170000
                    if file_type not in (0, stat.S_IFREG):
                        raise InstallError("release 压缩包包含非普通文件，已拒绝安装。")
                    if info.filename != expected_binary or found is not None:
                        raise InstallError("release 压缩包包含非预期文件，已拒绝安装。")
                    if info.file_size > MAX_DOWNLOAD_BYTES:
                        raise InstallError("PingCode MCP 程序超过 100 MiB 安全上限。")
                    found = bundle.read(info)
        else:
            with tarfile.open(fileobj=io.BytesIO(archive), mode="r:gz") as bundle:
                for member in bundle.getmembers():
                    if not safe_archive_member(member.name):
                        raise InstallError("release 压缩包包含不安全路径，已拒绝安装。")
                    if member.isdir():
                        continue
                    if not member.isfile() or member.name != expected_binary or found is not None:
                        raise InstallError("release 压缩包包含非预期文件，已拒绝安装。")
                    if member.size > MAX_DOWNLOAD_BYTES:
                        raise InstallError("PingCode MCP 程序超过 100 MiB 安全上限。")
                    stream = bundle.extractfile(member)
                    if stream is None:
                        raise InstallError("release 压缩包中的程序无法读取。")
                    found = stream.read(MAX_DOWNLOAD_BYTES + 1)
    except InstallError:
        raise
    except (tarfile.TarError, zipfile.BadZipFile, OSError, EOFError):
        raise InstallError("release 压缩包损坏或格式无效。") from None
    if not found:
        raise InstallError("release 压缩包内缺少预期的 PingCode MCP 程序。")
    if len(found) > MAX_DOWNLOAD_BYTES:
        raise InstallError("PingCode MCP 程序超过 100 MiB 安全上限。")
    return found


def verify_attestation(archive_path: Path) -> str:
    gh = shutil.which("gh")
    if not gh:
        return "跳过 provenance 验证（本机没有 GitHub CLI）；SHA-256 校验仍已完成。"
    command = [
        gh,
        "attestation",
        "verify",
        str(archive_path),
        "--repo",
        REPOSITORY_SLUG,
        "--signer-workflow",
        f"{REPOSITORY_SLUG}/.github/workflows/release.yml",
    ]
    try:
        result = subprocess.run(command, capture_output=True, text=True, timeout=60, check=False)
    except (OSError, subprocess.TimeoutExpired):
        raise InstallError("GitHub provenance 验证无法完成；已停止安装。") from None
    if result.returncode != 0:
        raise InstallError("GitHub provenance 验证失败；已停止安装。请更新 gh 后重试。")
    return "GitHub provenance 已验证。"


def install_dir(home: Path, goos: str) -> Path:
    if goos == "windows":
        local = os.environ.get("LOCALAPPDATA")
        base = Path(local) if local else home / "AppData" / "Local"
        return base / "Programs" / "PingCodeMCP"
    return home / ".local" / "bin"


def json_client_path(client: str, scope: str, workspace: Path | None, home: Path) -> tuple[Path, str]:
    if scope == "workspace" and workspace is None:
        raise InstallError("workspace 范围必须通过 --workspace 指定现有工作区目录。")
    if client == "kiro":
        return (home / ".kiro" / "settings" / "mcp.json", "mcpServers") if scope == "user" else (workspace / ".kiro" / "settings" / "mcp.json", "mcpServers")  # type: ignore[operator]
    if client == "cursor":
        return (home / ".cursor" / "mcp.json", "mcpServers") if scope == "user" else (workspace / ".cursor" / "mcp.json", "mcpServers")  # type: ignore[operator]
    if client == "vscode":
        return (home / ".copilot" / "mcp-config.json", "mcpServers") if scope == "user" else (workspace / ".mcp.json", "mcpServers")  # type: ignore[operator]
    raise InstallError("所选客户端使用非 JSON 配置。")


def server_config(client: str, executable: Path) -> dict[str, Any]:
    config: dict[str, Any] = {
        "command": str(executable),
        "args": [],
        "env": {"PINGCODE_READ_ONLY": "true"},
    }
    if client == "kiro":
        config["disabled"] = False
    elif client == "cursor":
        config["type"] = "stdio"
    return config


def load_json_config(path: Path) -> dict[str, Any]:
    if path.is_symlink():
        raise InstallError(f"配置目标是符号链接：{path}")
    if not path.exists():
        return {}
    if not path.is_file():
        raise InstallError(f"配置目标不是普通文件：{path}")
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeDecodeError, json.JSONDecodeError):
        raise InstallError(f"客户端 JSON 配置无法安全解析：{path}") from None
    if not isinstance(value, dict):
        raise InstallError(f"客户端 JSON 配置根节点必须是对象：{path}")
    return value


def json_config_state(path: Path, key: str, client: str, executable: Path) -> tuple[str, dict[str, Any]]:
    document = load_json_config(path)
    servers = document.get(key, {})
    if not isinstance(servers, dict):
        raise InstallError(f"配置字段 {key} 不是对象：{path}")
    desired = server_config(client, executable)
    existing = servers.get("pingcode")
    if existing is None:
        return "new", document
    if existing == desired:
        return "same", document
    return "conflict", document


def changed_field_names(current: Any, desired: dict[str, Any]) -> list[str]:
    if not isinstance(current, dict):
        return ["pingcode"]
    keys = set(current) | set(desired)
    return sorted(key for key in keys if current.get(key) != desired.get(key))


def write_atomic(path: Path, data: bytes) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    existing_mode = 0o600
    if path.is_symlink():
        raise InstallError(f"拒绝写入符号链接：{path}")
    if path.exists():
        if not path.is_file():
            raise InstallError(f"拒绝写入非普通文件：{path}")
        existing_mode = stat.S_IMODE(path.stat().st_mode)
    file_descriptor, temp_name = tempfile.mkstemp(prefix=".pingcode-", suffix=".tmp", dir=path.parent)
    try:
        if hasattr(os, "fchmod"):
            os.fchmod(file_descriptor, existing_mode)
        with os.fdopen(file_descriptor, "wb") as stream:
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temp_name, path)
    except OSError:
        try:
            os.unlink(temp_name)
        except OSError:
            pass
        raise InstallError(f"无法安全写入配置文件：{path}") from None


def json_config_bytes(document: dict[str, Any], key: str, client: str, executable: Path) -> bytes:
    servers = document.setdefault(key, {})
    if not isinstance(servers, dict):
        raise InstallError(f"配置字段 {key} 不是对象。")
    servers["pingcode"] = server_config(client, executable)
    return (json.dumps(document, ensure_ascii=False, indent=2) + "\n").encode("utf-8")


def codex_path(scope: str, workspace: Path | None, home: Path) -> Path:
    if scope == "user":
        return home / ".codex" / "config.toml"
    if workspace is None:
        raise InstallError("workspace 范围必须通过 --workspace 指定现有工作区目录。")
    return workspace / ".codex" / "config.toml"


def toml_server_state(path: Path, executable: Path) -> tuple[str, str]:
    if tomllib is None:
        raise InstallError("Codex TOML 配置需要 Python 3.11+ 的 tomllib。")
    if path.is_symlink():
        raise InstallError(f"配置目标是符号链接：{path}")
    if not path.exists():
        return "new", ""
    if not path.is_file():
        raise InstallError(f"配置目标不是普通文件：{path}")
    try:
        text = path.read_text(encoding="utf-8")
        parsed = tomllib.loads(text)
    except (OSError, UnicodeDecodeError, tomllib.TOMLDecodeError):
        raise InstallError(f"Codex TOML 配置无法安全解析：{path}") from None
    servers = parsed.get("mcp_servers", {})
    if not isinstance(servers, dict):
        raise InstallError("Codex 配置中的 mcp_servers 必须是 TOML table。")
    if re.search(r"(?m)^\s*mcp_servers\s*=", text):
        raise InstallError("Codex 配置使用 inline mcp_servers table；请先改成标准 TOML table。")
    existing = servers.get("pingcode")
    if existing is None:
        return "new", text
    desired = {"command": str(executable), "args": [], "env": {"PINGCODE_READ_ONLY": "true"}}
    return ("same" if existing == desired else "conflict"), text


def toml_server_block(executable: Path) -> str:
    command = json.dumps(str(executable), ensure_ascii=False)
    return (
        "[mcp_servers.pingcode]\n"
        f"command = {command}\n"
        "args = []\n"
        "env = { PINGCODE_READ_ONLY = \"true\" }\n"
    )


def updated_toml(text: str, executable: Path, replace: bool) -> bytes:
    if tomllib is None:
        raise InstallError("Codex TOML 配置需要 Python 3.11+ 的 tomllib。")
    state, _ = toml_server_state_from_text(text)
    if state == "same":
        return text.encode("utf-8")
    if state == "conflict" and not replace:
        raise InstallError("Codex 中已有不同的 pingcode MCP 配置；检查差异后使用 --replace。")
    updated = text
    if state == "conflict":
        pattern = re.compile(r"(?m)^\[mcp_servers(?:\.pingcode|\.\"pingcode\"|\.'pingcode')\]\s*(?:#.*)?\r?\n")
        match = pattern.search(updated)
        if not match:
            raise InstallError("无法安全替换 Codex 的 pingcode TOML table；请手动更新该项。")
        next_header = re.search(r"(?m)^\[[^\n]+\]\s*(?:#.*)?$", updated[match.end():])
        end = match.end() + next_header.start() if next_header else len(updated)
        updated = updated[:match.start()] + updated[end:]
    if updated and not updated.endswith("\n"):
        updated += "\n"
    if updated and not updated.endswith("\n\n"):
        updated += "\n"
    return (updated + toml_server_block(executable)).encode("utf-8")


def toml_server_state_from_text(text: str) -> tuple[str, dict[str, Any] | None]:
    if tomllib is None:
        raise InstallError("Codex TOML 配置需要 Python 3.11+ 的 tomllib。")
    try:
        parsed = tomllib.loads(text)
    except tomllib.TOMLDecodeError:
        raise InstallError("Codex TOML 配置无法安全解析。") from None
    servers = parsed.get("mcp_servers", {})
    if not isinstance(servers, dict):
        raise InstallError("Codex 配置中的 mcp_servers 必须是 TOML table。")
    if re.search(r"(?m)^\s*mcp_servers\s*=", text):
        raise InstallError("Codex 配置使用 inline mcp_servers table；请先改成标准 TOML table。")
    existing = servers.get("pingcode")
    if existing is None:
        return "new", None
    return "conflict", existing


def claude_scope(scope: str) -> str:
    return "user" if scope == "user" else "project"


def claude_existing(workspace: Path | None) -> bool:
    executable = shutil.which("claude")
    if not executable:
        raise InstallError("找不到 Claude Code CLI `claude`；请先安装客户端或使用手动配置步骤。")
    command = [executable, "mcp", "get", "pingcode"]
    try:
        result = subprocess.run(
            command,
            cwd=workspace,
            capture_output=True,
            text=True,
            timeout=15,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired):
        raise InstallError("无法检查 Claude Code 中的 pingcode 配置。") from None
    return result.returncode == 0


def claude_add(executable: Path, scope: str, workspace: Path | None) -> None:
    claude = shutil.which("claude")
    if not claude:
        raise InstallError("找不到 Claude Code CLI `claude`。")
    payload = {
        "type": "stdio",
        "command": str(executable),
        "args": [],
        "env": {"PINGCODE_READ_ONLY": "true"},
    }
    command = [claude, "mcp", "add-json", "pingcode", json.dumps(payload, separators=(",", ":")), "--scope", claude_scope(scope)]
    try:
        result = subprocess.run(command, cwd=workspace, capture_output=True, text=True, timeout=30, check=False)
    except (OSError, subprocess.TimeoutExpired):
        raise InstallError("Claude Code 配置命令无法完成。") from None
    if result.returncode != 0:
        raise InstallError("Claude Code 未接受 MCP 配置；已保留原配置。请按 references/ide-setup.md 手动配置。")


def client_configuration(args: argparse.Namespace, executable: Path, home: Path) -> tuple[str, Path | None, dict[str, Any] | None]:
    if args.client in {"kiro", "cursor", "vscode"}:
        path, key = json_client_path(args.client, args.scope, args.workspace, home)
        state, document = json_config_state(path, key, args.client, executable)
        if state == "conflict" and not (args.replace or args.dry_run):
            servers = document.get(key, {})
            current = servers.get("pingcode") if isinstance(servers, dict) else None
            changed = changed_field_names(current, server_config(args.client, executable))
            fields = ", ".join(changed)
            raise InstallError(f"{args.client} 中已有不同的 pingcode MCP 配置（差异字段：{fields}）；确认后使用 --replace。路径：{path}")
        return state, path, {"document": document, "key": key}
    if args.client == "codex":
        path = codex_path(args.scope, args.workspace, home)
        state, text = toml_server_state(path, executable)
        if state == "conflict" and not (args.replace or args.dry_run):
            raise InstallError(f"Codex 中已有不同的 pingcode MCP 配置；检查差异后使用 --replace。路径：{path}")
        return state, path, {"text": text}
    if args.client == "claude":
        exists = claude_existing(args.workspace)
        if exists and not (args.replace or args.dry_run):
            raise InstallError("Claude Code 中已有 pingcode MCP 配置；如确认替换，请加 --replace。")
        return ("conflict" if exists else "new"), None, None
    raise InstallError("未知 MCP 客户端。")


def plan_paths(args: argparse.Namespace, goos: str, home: Path) -> tuple[Path, Path | None, str | None]:
    destination = install_dir(home, goos) / binary_name(goos)
    if args.client == "claude":
        return destination, None, None
    if args.client == "codex":
        return destination, codex_path(args.scope, args.workspace, home), "TOML"
    config_path, key = json_client_path(args.client, args.scope, args.workspace, home)
    return destination, config_path, f"JSON ({key})"


def prompt_to_continue(args: argparse.Namespace) -> None:
    if args.yes:
        return
    if not sys.stdin.isatty():
        raise InstallError("非交互安装需要显式传入 --yes；dry-run 不需要确认。")
    try:
        answer = input("确认下载、安装程序并写入所选 IDE 配置？[y/N] ").strip().lower()
    except (EOFError, KeyboardInterrupt):
        raise InstallError("用户取消安装；没有写入配置。") from None
    if answer not in {"y", "yes"}:
        raise InstallError("用户取消安装；没有写入配置。")


def verify_mcp(executable: Path, timeout: int = 20) -> None:
    """Initialize and list tools without using PingCode credentials or APIs."""
    from queue import Empty, Queue

    env = {key: value for key, value in os.environ.items() if not key.startswith("PINGCODE_")}
    env["PINGCODE_READ_ONLY"] = "true"
    env["PINGCODE_ALLOWED_PROJECT_IDS"] = ""
    process: subprocess.Popen[str] | None = None
    try:
        process = subprocess.Popen(
            [str(executable)],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            text=True,
            bufsize=1,
            env=env,
        )
    except OSError:
        raise InstallError("无法启动安装后的 PingCode MCP 程序。") from None
    if process.stdin is None or process.stdout is None:
        process.kill()
        raise InstallError("无法建立 MCP 标准输入输出通道。")

    output: Queue[str | None] = Queue()

    def collect_stdout() -> None:
        assert process is not None and process.stdout is not None
        for output_line in process.stdout:
            output.put(output_line)
        output.put(None)

    reader = threading.Thread(target=collect_stdout, daemon=True)
    reader.start()

    def send(message: dict[str, Any]) -> None:
        assert process is not None and process.stdin is not None
        process.stdin.write(json.dumps(message) + "\n")
        process.stdin.flush()

    def await_response(request_id: int, wait_seconds: int) -> dict[str, Any]:
        deadline = time.monotonic() + wait_seconds
        while True:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                raise InstallError("MCP 本地初始化超时；未调用 PingCode API。")
            try:
                line = output.get(timeout=remaining)
            except Empty:
                raise InstallError("MCP 本地初始化超时；未调用 PingCode API。") from None
            if line is None:
                raise InstallError("MCP 程序在返回协议响应前退出。")
            try:
                response = json.loads(line)
            except json.JSONDecodeError:
                raise InstallError("MCP 程序向 stdout 写入了非协议内容。") from None
            if not isinstance(response, dict):
                raise InstallError("MCP 程序返回了无效协议消息。")
            response_id = response.get("id")
            if isinstance(response_id, int) and response_id == request_id:
                return response

    try:
        send(
            {
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {},
                    "clientInfo": {"name": "pingcode-mcp-setup", "version": "1"},
                },
            }
        )
        initialize_response = await_response(1, timeout)
        if "error" in initialize_response:
            raise InstallError("MCP initialize 未成功；未调用 PingCode API。")
        send({"jsonrpc": "2.0", "method": "notifications/initialized"})
        send({"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}})
        tools_response = await_response(2, timeout)
        if "error" in tools_response:
            raise InstallError("MCP tools/list 未成功；未调用 PingCode API。")
        tools_result = tools_response.get("result", {})
        names = {
            item.get("name")
            for item in tools_result.get("tools", [])
            if isinstance(item, dict)
        }
        if names != EXPECTED_TOOLS:
            raise InstallError("MCP tools/list 返回的工具集合与当前服务规格不一致。")
        process.stdin.close()
        try:
            exit_code = process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()
            raise InstallError("MCP 服务关闭超时；未调用 PingCode API。") from None
        if exit_code != 0:
            raise InstallError("MCP 本地初始化失败；未调用 PingCode API。")
    except InstallError:
        if process.poll() is None:
            process.kill()
            process.wait()
        raise
    except OSError:
        if process.poll() is None:
            process.kill()
            process.wait()
        raise InstallError("MCP 标准输入输出通道中断。") from None
    finally:
        if process.poll() is None:
            process.kill()
            process.wait()
        if process.stdin is not None and not process.stdin.closed:
            process.stdin.close()
        reader.join(timeout=1)
        if process.stdout is not None and not process.stdout.closed:
            process.stdout.close()


def verify_binary_version(executable: Path, expected: str) -> None:
    try:
        result = subprocess.run(
            [str(executable), "--version"],
            capture_output=True,
            text=True,
            timeout=10,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired):
        raise InstallError("无法读取下载程序的版本。") from None
    if result.returncode != 0 or result.stdout.strip() != expected:
        raise InstallError("下载程序的版本与选定版本不匹配。")


def go_version_ok() -> None:
    go = shutil.which("go")
    if not go:
        raise InstallError("源码构建需要 Go 1.25+；本机未找到 go。")
    try:
        result = subprocess.run([go, "version"], capture_output=True, text=True, timeout=10, check=False)
    except (OSError, subprocess.TimeoutExpired):
        raise InstallError("无法读取 Go 版本。") from None
    match = re.search(r"\bgo(\d+)\.(\d+)(?:\.(\d+))?\b", result.stdout)
    if result.returncode != 0 or not match or tuple(map(int, match.groups(default="0"))) < (1, 25, 0):
        raise InstallError("源码构建需要 Go 1.25 或更新版本。")


def source_archive(ref: str) -> bytes:
    return request_bytes(f"https://api.github.com/repos/{REPOSITORY_SLUG}/tarball/{ref}")


def extract_source(archive: bytes, target: Path) -> Path:
    target.mkdir(parents=True, exist_ok=True)
    try:
        with tarfile.open(fileobj=io.BytesIO(archive), mode="r:gz") as bundle:
            root_names: set[str] = set()
            extracted_size = 0
            for member in bundle.getmembers():
                if not safe_archive_member(member.name):
                    raise InstallError("GitHub 源码归档包含不安全路径。")
                if member.isdir():
                    (target / member.name).mkdir(parents=True, exist_ok=True)
                    root_names.add(member.name.split("/", 1)[0])
                    continue
                if not member.isfile():
                    raise InstallError("GitHub 源码归档包含非普通文件。")
                if member.size < 0 or member.size > MAX_DOWNLOAD_BYTES:
                    raise InstallError("GitHub 源码文件超过 100 MiB 安全上限。")
                extracted_size += member.size
                if extracted_size > MAX_SOURCE_BYTES:
                    raise InstallError("GitHub 源码归档解压后超过 500 MiB 安全上限。")
                root_names.add(member.name.split("/", 1)[0])
                destination = target / member.name
                destination.parent.mkdir(parents=True, exist_ok=True)
                source = bundle.extractfile(member)
                if source is None:
                    raise InstallError("GitHub 源码归档中的文件无法读取。")
                destination.write_bytes(source.read(MAX_DOWNLOAD_BYTES + 1))
            if len(root_names) != 1:
                raise InstallError("GitHub 源码归档的目录结构不符合预期。")
            source_root = target / next(iter(root_names))
    except InstallError:
        raise
    except (tarfile.TarError, OSError, EOFError):
        raise InstallError("GitHub 源码归档损坏或格式无效。") from None
    if not (source_root / "go.mod").is_file() or not (source_root / "cmd" / "pingcode-mcp").is_dir():
        raise InstallError("GitHub 源码归档缺少 Go 模块或 MCP 命令入口。")
    return source_root


def source_binary(ref: str, goos: str, goarch: str, temp: Path) -> Path:
    archive = source_archive(ref)
    source_root = extract_source(archive, temp / "source")
    output = temp / binary_name(goos)
    go = shutil.which("go")
    if not go:
        raise InstallError("源码构建需要 Go 1.25+。")
    env = os.environ.copy()
    env.update({"GOOS": goos, "GOARCH": goarch, "CGO_ENABLED": "0", "PINGCODE_READ_ONLY": "true"})
    version_label = ref if re.fullmatch(r"v\d+\.\d+\.\d+", ref) else "0.1.0-dev"
    try:
        result = subprocess.run(
            [
                go,
                "build",
                "-trimpath",
                "-ldflags",
                f"-X main.version={version_label}",
                "-o",
                str(output),
                "./cmd/pingcode-mcp",
            ],
            cwd=source_root,
            env=env,
            capture_output=True,
            text=True,
            timeout=180,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired):
        raise InstallError("源码构建无法完成。") from None
    if result.returncode != 0 or not output.is_file():
        raise InstallError("源码构建失败；未安装程序。")
    return output


def install_binary(staged: Path, destination: Path) -> tuple[bytes | None, int | None]:
    destination.parent.mkdir(parents=True, exist_ok=True)
    previous: bytes | None = None
    previous_mode: int | None = None
    if destination.is_symlink():
        raise InstallError(f"拒绝覆盖符号链接：{destination}")
    if destination.exists():
        if not destination.is_file():
            raise InstallError(f"拒绝覆盖非普通文件：{destination}")
        previous = destination.read_bytes()
        previous_mode = stat.S_IMODE(destination.stat().st_mode)
    mode = previous_mode if previous_mode is not None else 0o755
    try:
        fd, temp_name = tempfile.mkstemp(prefix=".pingcode-mcp-", dir=destination.parent)
        with os.fdopen(fd, "wb") as stream:
            stream.write(staged.read_bytes())
            stream.flush()
            os.fsync(stream.fileno())
        os.chmod(temp_name, mode)
        os.replace(temp_name, destination)
    except OSError:
        try:
            os.unlink(temp_name)
        except (OSError, UnboundLocalError):
            pass
        raise InstallError(f"无法安装程序到：{destination}") from None
    return previous, previous_mode


def restore_binary(destination: Path, previous: bytes | None, previous_mode: int | None) -> None:
    try:
        if previous is None:
            destination.unlink(missing_ok=True)
        else:
            fd, temp_name = tempfile.mkstemp(prefix=".pingcode-mcp-rollback-", dir=destination.parent)
            with os.fdopen(fd, "wb") as stream:
                stream.write(previous)
            os.chmod(temp_name, previous_mode or 0o755)
            os.replace(temp_name, destination)
    except OSError:
        raise InstallError("配置写入失败，且自动回滚二进制失败；请检查目标安装目录。") from None


def apply_client_config(
    args: argparse.Namespace,
    executable: Path,
    home: Path,
    state: str,
    details: dict[str, Any] | None,
) -> None:
    if state == "same":
        return
    if args.client == "claude":
        claude_add(executable, args.scope, args.workspace)
        return
    if details is None:
        raise InstallError("缺少客户端配置上下文。")
    path = Path(details["path"])
    if args.client == "codex":
        _, text = toml_server_state(path, executable)
        data = updated_toml(text, executable, args.replace)
    else:
        _, key = json_client_path(args.client, args.scope, args.workspace, home)
        current_state, document = json_config_state(path, key, args.client, executable)
        if current_state == "conflict" and not args.replace:
            raise InstallError(f"{args.client} 配置在安装期间发生变化；重新预览后再尝试。路径：{path}")
        if current_state == "same":
            return
        data = json_config_bytes(document, key, args.client, executable)
    write_atomic(path, data)


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="下载并配置 PingCode MCP stdio 程序。")
    parser.add_argument("--client", choices=("kiro", "codex", "claude", "cursor", "vscode"), required=True)
    parser.add_argument("--scope", choices=("user", "workspace"), default="user")
    parser.add_argument("--workspace", type=Path, help="workspace 根目录；--scope workspace 时必填")
    parser.add_argument("--version", help="固定稳定版本，例如 v1.2.3")
    parser.add_argument("--dry-run", action="store_true", help="只展示计划，不联网、不写文件")
    parser.add_argument("--build-from-source", action="store_true", help="显式从 GitHub 源码构建；需要 Go 1.25+")
    parser.add_argument("--replace", action="store_true", help="允许替换现有 pingcode 配置或安装程序")
    parser.add_argument("--yes", action="store_true", help="非交互执行时确认本次安装")
    return parser.parse_args(argv)


def run_install(args: argparse.Namespace) -> int:
    if sys.version_info < (3, 11) or tomllib is None:
        raise InstallError("安装脚本需要 Python 3.11 或更新版本。")
    if args.version:
        validate_tag(args.version)
    if args.scope == "workspace":
        if args.workspace is None or not args.workspace.exists() or not args.workspace.is_dir():
            raise InstallError("--scope workspace 需要 --workspace 指向一个已存在的工作区目录。")
        args.workspace = args.workspace.resolve()
    elif args.workspace is not None:
        args.workspace = args.workspace.resolve()
    goos, goarch = target_platform()
    home = Path.home()
    destination, config_path, config_format = plan_paths(args, goos, home)
    if args.dry_run:
        state, config_path, _ = client_configuration(args, destination, home)
        version_text = args.version or ("源码分支 main" if args.build_from_source else "最新正式版本（dry-run 不联网解析）")
        print("安装计划：")
        print(f"  客户端：{args.client} ({args.scope})")
        print(f"  平台：{goos}/{goarch}")
        print(f"  版本：{version_text}")
        print(f"  程序：{destination}")
        if config_path:
            print(f"  配置：{config_path} [{config_format}]")
        else:
            print(f"  配置：Claude Code CLI ({claude_scope(args.scope)} scope)")
        if state == "same":
            print("  pingcode 配置：已匹配。")
        elif state == "conflict":
            print("  pingcode 配置：存在冲突；正式安装需显式加 --replace。")
        if destination.exists():
            print("  安装程序：目标已存在；正式安装需显式加 --replace。")
        print("dry-run 未联网、下载、解包或写入文件。")
        return 0

    if destination.exists() and not args.replace:
        raise InstallError(f"安装目录已存在程序；如确认替换，请加 --replace。路径：{destination}")
    state, config_path, config_details = client_configuration(args, destination, home)
    print("安装计划：")
    print(f"  客户端：{args.client} ({args.scope})")
    print(f"  平台：{goos}/{goarch}")
    print(f"  目标程序：{destination}")
    if config_path:
        print(f"  配置文件：{config_path}")
    else:
        print(f"  配置目标：Claude Code CLI ({claude_scope(args.scope)} scope)")
    print("  环境变量：仅设置 PINGCODE_READ_ONLY=true；不处理凭据。")
    if state == "same":
        print("  pingcode 配置：已匹配，无需改写。")
    elif state == "conflict":
        print("  pingcode 配置：已存在；--replace 将替换此项。")
    if args.build_from_source:
        go_version_ok()
        print(f"  来源：GitHub 源码 {args.version or 'main'}（需要 Go 1.25+）。")
        ref = args.version or "main"
    else:
        release = get_release(args.version)
        tag = release["tag_name"]
        archive_name = release_asset_name(tag, goos, goarch)
        archive_url = asset_url(release, archive_name)
        sums_url = asset_url(release, "SHA256SUMS")
        print(f"  Release：{tag} / {archive_name}")
        print("  校验：SHA-256；若本机有 gh，还会校验 GitHub provenance。")
    if destination.exists() and args.replace:
        print("  将替换已存在的本地程序。")
    prompt_to_continue(args)

    with tempfile.TemporaryDirectory(prefix="pingcode-mcp-setup-") as temp_name:
        temp = Path(temp_name)
        if args.build_from_source:
            staged = source_binary(ref, goos, goarch, temp)
            provenance_status = "源码构建模式未使用 Release provenance。"
            expected_version = ref if re.fullmatch(r"v\d+\.\d+\.\d+", ref) else "0.1.0-dev"
        else:
            archive = request_bytes(archive_url)
            manifest = request_bytes(sums_url)
            verify_checksum(archive, manifest, archive_name)
            staged_bytes = unpack_expected_binary(archive, archive_name, binary_name(goos))
            staged = temp / binary_name(goos)
            staged.write_bytes(staged_bytes)
            staged.chmod(0o755)
            # Verify the attestation for the exact downloaded archive bytes.
            archive_path = temp / archive_name
            archive_path.write_bytes(archive)
            provenance_status = verify_attestation(archive_path)
            expected_version = tag
        verify_binary_version(staged, expected_version)
        verify_mcp(staged)
        previous, previous_mode = install_binary(staged, destination)
        if config_details is not None:
            config_details["path"] = str(config_path)
        try:
            apply_client_config(args, destination, home, state, config_details)
        except InstallError:
            restore_binary(destination, previous, previous_mode)
            raise
    print("安装完成。")
    print(f"  程序：{destination}")
    if config_path:
        print(f"  配置：{config_path}")
    print(f"  {provenance_status}")
    print("  本地 MCP initialize/tools/list 验证通过；未调用 PingCode API。")
    print("  凭据如有需要，请只通过本机环境变量提供；只读默认保持开启。")
    return 0


def main(argv: list[str] | None = None) -> int:
    try:
        args = parse_args(argv)
        return run_install(args)
    except InstallError as exc:
        print(f"安装失败：{exc}", file=sys.stderr)
        return 1
    except KeyboardInterrupt:
        print("安装已取消。", file=sys.stderr)
        return 130


if __name__ == "__main__":
    raise SystemExit(main())
