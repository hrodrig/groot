# Plan 1.3.0 — archive trust

**Status:** open  
**Target release:** **`v1.3.0`**  
**Roadmap:** **#49**, **#100**–**#107** — see [ROADMAP.md](../ROADMAP.md)  
**Source:** audit `groot-all/groot/audits/auditoria-claude-20261015.txt` (static review of v1.2.0), plus the `archive_sha256` gap found while checking that review  
**Delivery:** topic branch → PR into `develop` per item (or a tight pair). Do not mix the `VERSION` bump into these PRs. `chore/release-1.3.0` is a later, separate PR.

---

## Goal

A collect that leaves the machine is safe to keep and safe to hand to notify: secrets in the capture are redacted by default, credential-bearing `config view` output is not stored raw, the archive is private and complete, its checksum is published beside the tar, mail auth cannot fall back to cleartext, the LLM paste cannot be closed by a log line, and a failed upload is visible in the exit code before anyone is told the run succeeded.

Kind E2E on the PR is the cluster that proves collect still does that. There is no other Kubernetes in the default path.

## Merge order

1. **#49** — `test-e2e-kind` fails the PR (`continue-on-error` removed).
2. **#100** — redaction patterns, fail-closed pass, scope beyond `*.log`, default `redact_secrets: true`.
3. **#101** — `config view` credential redaction.
4. **#102** — `0700` directories, `0600` files and `.tar.gz`.
5. **#103** — propagate `DirToTarGz` `Close` errors.
6. **#107** — `archive_sha256` sidecar (depends on a close that fails closed).
7. **#104** — STARTTLS required on SMTP port 587.
8. **#105** — LLM excerpt fence.
9. **#106** — upload, then notify; non-zero exit if any upload fails.

## Success criteria

- [ ] **#49** A red `make test-e2e-kind` in GitHub Actions fails the workflow. The job still runs on pull requests to `develop` and `main`.
- [ ] **#100** With the default config, `Authorization: Bearer eyJ…`, `{"token":"abc"}`, `password: "hunter2 extra"`, and `DB_PASSWORD=x` do not leave a usable secret. The key name survives (`DB_PASSWORD=[REDACTED]`). `resources.txt` does not keep `env[].value`. A redaction error aborts before archive and upload. `redact_secrets: false` still opts out.
- [ ] **#101** `extra_kubectl` `config view` output in the capture has token, client key, client certificate, password, and exec secrets redacted.
- [ ] **#102** New capture directories are `0700`. New capture files and the `.tar.gz` are `0600`.
- [ ] **#103** A `Close` error on the tar, gzip, or destination file fails the collect. A truncated archive is not uploaded.
- [ ] **#107** After a successful collect, a sidecar next to the `.tar.gz` contains `archive_sha256` for that file. The hash is not expected inside the tar.
- [ ] **#104** SMTP on port 587 does not send AUTH or message data when STARTTLS is absent.
- [ ] **#105** An excerpt that contains a triple-backtick fence cannot close the LLM Markdown block early.
- [ ] **#106** Upload runs before success notify. If any provider fails, the process exits non-zero and notify describes that failure. A notify failure after a successful upload still uses the notify exit code. `upload.continue_on_error` still tries the remaining providers.
- [ ] SPEC, sample config, and `[Unreleased]` CHANGELOG match the behavior above, with `(1.3.x #N)` cites. `VERSION` stays **1.2.0** until `chore/release-1.3.0`.

## SPEC surfaces

| Item | Contract today | Change |
|------|----------------|--------|
| #100 | `collection.redact_secrets` default `false`; `*.log` only | Default `true`; text captures including `resources.txt`; fail the run if the pass errors |
| #101 | `config view` is an allowlisted read | Stored output redacts credential fields |
| #102 | Capture dirs `0755`, files `0644`, tar via `os.Create` | `0700` / `0600` |
| #103 | Close errors discarded | Close errors fail the archive step (exit 3 family) |
| #107 | `#81` promised `archive_sha256` in the manifest | Sidecar beside the tar; document that the in-tar manifest has no self-hash |
| #104 | Port 587 continues when STARTTLS is not advertised | Refuse cleartext AUTH |
| #105 | LLM excerpts sit in a triple-backtick fence | Fence cannot be terminated by excerpt text |
| #106 | Notify, then upload; upload errors stay exit 0 | Upload, then notify; upload failure is its own non-zero exit |
| #49 | Kind job is `continue-on-error: true` | Job failure fails the workflow |

## Out of scope

Rejected for this release, and not parked as follow-up work:

- `arcread` re-reading the gzip for every member.
- Lowering the 16 GiB decompressed cap or the 100 000 file cap.
- Pinning GitHub Actions to SHAs, pinning `govulncheck@latest` / `grype:latest`, or checksum-pinning the kind/kubectl `curl` installs.

Still elsewhere:

- **#45** `--dry-run-redact` only.
- **#32** multi-cluster and **#56** `groot diff` (after this cut).
- **#99** archive MCP, target **v1.4.0**, after redaction is on by default.
- `chore/release-1.3.0` (`VERSION`, CHANGELOG cut, README badge, `make man-sync`, port sync, VHS `docs/demo.gif`).

## Release checklist (later, not this plan)

Do not start until #49 and #100–#107 are on `develop` and `make release-check STRICT_RELEASE=1` is green. Follow **groot-release**: solo `chore/release-1.3.0` → `develop`, then PR `develop` → `main`, then annotated tag `v1.3.0` on `main`.
