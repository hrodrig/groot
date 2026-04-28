# Security policy

## Scope

This policy covers the GROOT CLI, its configuration handling, optional notification integrations (Slack, Telegram, Teams), and the published container image. GROOT runs `kubectl` and writes diagnostics to disk; treat output archives as potentially sensitive (see [README](README.md)).

## Supported versions

We support the **latest release** tagged on `main` and meaningful security fixes on the current development branch (`develop`). Versions follow [semantic versioning](https://semver.org/) (MAJOR.MINOR.PATCH).

| Version | Supported |
| ------- | --------- |
| Latest release (see [releases](https://github.com/hrodrig/groot/releases)) | Yes |
| Older releases | No — upgrade to latest |

Security fixes ship as patch releases when applicable. Use the newest release to get fixes.

## Code scanning (CodeQL)

This repository uses **[GitHub CodeQL](https://docs.github.com/en/code-security/code-scanning/introduction-to-code-scanning/about-code-scanning-with-codeql)** (code scanning) on pushes and pull requests when the workflow is enabled. Alerts appear under the repo’s **Security** tab.

We triage CodeQL results like any other signal: **true positives** are fixed in code (often shipped as patch releases); **false positives or intentional behavior** may be dismissed with a short rationale in GitHub. Examples of rules that matter for a CLI like GROOT include unsafe handling of user-controlled paths, injection-style patterns, and **clear-text logging of sensitive information** (e.g. credentials or connection secrets in logs or notification payloads) — avoid introducing those; follow existing patterns for redaction and safe logging.

If you open a PR, fix or discuss new CodeQL alerts before merge when they are valid.

## Reporting a vulnerability

**Do not open a public issue** for undisclosed security vulnerabilities.

- **Preferred:** [Report a vulnerability](https://github.com/hrodrig/groot/security/advisories/new) via GitHub Security Advisories (private to maintainers).
- **Alternatively:** Contact the maintainer through options on [github.com/hrodrig](https://github.com/hrodrig). Include description, steps to reproduce, affected versions (if known), and impact.

## What to expect

- Acknowledgment as soon as practical.
- Investigation, fix timeline, and updates on critical issues.
- Credit in the advisory or release notes if you want it; anonymous disclosure respected if you ask.
- Brief explanation if we decline or defer (e.g. out of scope).

Thank you for helping keep GROOT and its users safe.
