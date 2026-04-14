# Security Policy

## Supported versions

This project is pre-1.0. Only the latest release receives security fixes.

## Reporting a vulnerability

**Please do not open a public issue for security vulnerabilities.**

Instead, use GitHub's [private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability) on this repository. You can expect:

- An acknowledgment within 7 days.
- A triage and severity assessment within 14 days.
- A fix or mitigation timeline based on severity.

## Scope

doit is a local terminal application. Please report:

- Path traversal or arbitrary file write outside the configured `--repo`.
- Command injection into the `git` exec boundary (e.g. a crafted task title that breaks out of arguments).
- Any code path that causes the app to run a non-allow-listed git command (e.g. `push`, `pull`, `fetch`, `reset`).
- Terminal-escape injection via rendered task content (malicious markdown that hijacks the terminal).

Out of scope:
- Issues that require an attacker who already has write access to the user's filesystem or git repo.
- Denial of service via hand-crafted markdown in the user's own file.
