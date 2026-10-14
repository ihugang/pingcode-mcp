# Installer Download Proxy

## ADDED Requirements

### Requirement: Use Runtime Environment Proxy for Release Downloads

安装器 SHALL route GitHub release metadata and asset download requests through the proxy settings available to the installer process. It SHALL prefer protocol-specific `HTTP_PROXY`/`HTTPS_PROXY` settings, fall back to `ALL_PROXY` when no protocol-specific proxy is set, and respect proxy bypass settings such as `NO_PROXY`. If the selected proxy protocol is unsupported, the installer MUST stop rather than silently make a direct request. Proxy values MUST NOT be printed, stored in IDE configuration, or copied into the Skill.

#### Scenario: Download through a configured proxy

- **GIVEN** the installer process has a protocol-specific HTTP(S) proxy or an HTTP(S) `ALL_PROXY` fallback configured in its environment
- **WHEN** the installer fetches release metadata, an archive, or a checksum manifest
- **THEN** the request uses the proxy selected for that URL protocol
- **AND** proxy bypass rules are honored
- **AND** the proxy value is not included in output or configuration files

#### Scenario: Unsupported proxy is configured

- **GIVEN** the selected proxy uses an unsupported protocol such as SOCKS and the target is not covered by `NO_PROXY`
- **WHEN** the installer fetches a GitHub release resource
- **THEN** the installer stops with a safe explanation
- **AND** it does not silently connect directly
- **AND** it does not reveal the proxy value

#### Scenario: No proxy is configured

- **GIVEN** the installer process has no applicable proxy configured
- **WHEN** the installer fetches a GitHub release resource
- **THEN** the download uses the normal direct connection path
- **AND** checksum and GitHub host validation remain enabled
