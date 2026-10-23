# PastureStack Webhook Automation Service

> **PastureStack modification notice:** the current tree follows the preserved upstream `v0.9.15` boundary and contains a consolidated maintenance change for neutral naming, current-toolchain builds, and security hardening. The inherited history remains authoritative for upstream work.

Webhook Automation Service provides the webhook receiver API used by the preserved v1.6 control plane. It supports service scaling, host scaling, service upgrades after registry events, and controlled forwarding to an application service.

PastureStack is an independent community effort to preserve, audit, and modernize the Rancher 1.6 ecosystem. It is not affiliated with or endorsed by Rancher Labs or SUSE.

**Upstream:** [`rancher/webhook-service`](https://github.com/rancher/webhook-service), relevant branch/tag `v1.6` / `v0.9.15`, commit `5d68737e9c5edafc70a4963ffca1466e0b95c708`. This fork preserves upstream history, authorship, dates, tags, and the Apache-2.0 license. PastureStack claims authorship only for its own changes.

The maintenance source builds `v0.10.6` with Go 1.27.3, including the
[crypto/x509 security fix](https://go.dev/doc/devel/release#go1.27.3).
Use the [latest published GitHub Release](https://github.com/PastureStack/webhook-automation-service/releases/latest)
and its exact asset checksum for installation. A source version is not proof
of publication or Server acceptance. Existing tags and release assets remain
immutable; Server images must explicitly include this component.
See the [Receiver schema release notes](.github/release-notes/v0.10.3.md)
for the role-aware API and its verification boundary.

## Runtime boundary

The executable is `webhook-automation-service`. It binds to `127.0.0.1:8085` by default and accepts only the RSA public key needed to verify RS256 webhook tokens. It does not read or retain the control-plane private key.

The four inherited driver identifiers and `/v1-webhooks` routes remain stable because the Web Console and orchestration engine use them as wire contracts. New configuration uses `PASTURESTACK_*` environment names; the required inherited aliases are documented in [COMPATIBILITY.md](COMPATIBILITY.md).

Security changes include strict RS256 verification without the retired JWT dependency, bounded request and response bodies, request timeouts, an exact control-plane-origin policy enforced at the network boundary, redirect and ambient-proxy refusal for credentialed forwarding, sensitive-header filtering, and removal of committed test key fixtures. See [SECURITY.md](SECURITY.md).

## Build and test

The source uses Go Modules and a checked-in vendor tree for reproducible builds. On Linux with Go 1.27.3, `bash`, `tar`, `xz`, `curl`, OpenSSL, and a C toolchain for race tests:

```sh
git clone https://github.com/PastureStack/webhook-automation-service.git
cd webhook-automation-service
make validate
make test
make build
make integration-test
```

The `webhook-automation-service-0.10.6-linux-amd64.tar.xz` archive produced by
this source contains the executable plus compatibility, source, notice, and
composite license files. Formal producer packaging records its SHA-256;
installation must verify the matching published immutable asset rather than
reusing a previous release's checksum. An unpublished source archive must not
be treated as a public release or native acceptance result.

See [COMPATIBILITY.md](COMPATIBILITY.md), [ORIGIN.md](ORIGIN.md), [MODIFICATIONS.md](MODIFICATIONS.md), and [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md).

## License and attribution

The inherited project remains licensed under the [Apache License 2.0](LICENSE). Copyright and attribution for inherited and vendored work remain with their respective authors and contributors. No upstream work is presented as original PastureStack work.

## October 2026 security maintenance

Maintained Go builds select the patched Go 1.27.3 compiler without raising the existing Go language requirement. Release artifacts must pass the repository's existing checks before publication.
