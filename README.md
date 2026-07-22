# PastureStack Webhook Automation Service

> **PastureStack modification notice:** the current tree follows the preserved upstream `v0.9.15` boundary and contains a consolidated maintenance change for neutral naming, current-toolchain builds, and security hardening. The inherited history remains authoritative for upstream work.

Webhook Automation Service provides the webhook receiver API used by the preserved v1.6 control plane. It supports service scaling, host scaling, service upgrades after registry events, and controlled forwarding to an application service.

PastureStack is an independent community effort to preserve, audit, and modernize the Rancher 1.6 ecosystem. It is not affiliated with or endorsed by Rancher Labs or SUSE.

**Upstream:** [`rancher/webhook-service`](https://github.com/rancher/webhook-service), relevant branch/tag `v1.6` / `v0.9.15`, commit `5d68737e9c5edafc70a4963ffca1466e0b95c708`. This fork preserves upstream history, authorship, dates, tags, and the Apache-2.0 license. PastureStack claims authorship only for its own changes.

## Runtime boundary

The executable is `webhook-automation-service`. It binds to `127.0.0.1:8085` by default and accepts only the RSA public key needed to verify RS256 webhook tokens. It does not read or retain the control-plane private key.

The four inherited driver identifiers and `/v1-webhooks` routes remain stable because the Web Console and orchestration engine use them as wire contracts. New configuration uses `PASTURESTACK_*` environment names; the required inherited aliases are documented in [COMPATIBILITY.md](COMPATIBILITY.md).

Security changes include strict RS256 verification without the retired JWT dependency, bounded request and response bodies, request timeouts, an exact control-plane-origin policy enforced at the network boundary, redirect and ambient-proxy refusal for credentialed forwarding, sensitive-header filtering, and removal of committed test key fixtures. See [SECURITY.md](SECURITY.md).

## Build and test

The source uses its exact vendored dependency tree in GOPATH mode. On Linux with Go 1.26, `bash`, `tar`, `xz`, `curl`, OpenSSL, and a C toolchain for race tests:

```sh
mkdir -p "${GOPATH}/src/github.com/PastureStack"
git clone https://github.com/PastureStack/webhook-automation-service.git \
  "${GOPATH}/src/github.com/PastureStack/webhook-automation-service"
cd "${GOPATH}/src/github.com/PastureStack/webhook-automation-service"
make validate
make test
make build
make integration-test
```

`make package` creates the deterministic flat GitHub Release asset `webhook-automation-service-0.10.1-linux-amd64.tar.xz`. The archive contains the executable plus compatibility, source, notice, and composite license files. The Server release verifies its SHA-256 digest before installation, so operators do not need an artifact mirror.

See [COMPATIBILITY.md](COMPATIBILITY.md), [ORIGIN.md](ORIGIN.md), [MODIFICATIONS.md](MODIFICATIONS.md), and [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md).

## License and attribution

The inherited project remains licensed under the [Apache License 2.0](LICENSE). Copyright and attribution for inherited and vendored work remain with their respective authors and contributors. No upstream work is presented as original PastureStack work.
