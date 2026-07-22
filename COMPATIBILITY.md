# Compatibility Contract

The migration preserves the v1.6 surface consumed by the orchestration engine and Web Console:

- `GET /v1-webhooks` and the inherited schemas, receivers, and endpoint routes;
- the `scaleService`, `scaleHost`, `serviceUpgrade`, and `forwardPost` driver identifiers;
- loopback port `8085`;
- `RSA_PUBLIC_KEY_CONTENTS` as a secondary alias for `PASTURESTACK_API_PUBLIC_KEY_CONTENTS`;
- `CATTLE_URL`, `CATTLE_ACCESS_KEY`, and `CATTLE_SECRET_KEY` as secondary aliases for `PASTURESTACK_API_URL`, `PASTURESTACK_API_ACCESS_KEY`, and `PASTURESTACK_API_SECRET_KEY`; and
- the inherited `go-rancher` API types, label keys, placeholder image value, and `/v2-beta` API behavior needed by the preserved wire contract.

The legacy product strings above are compatibility identifiers, dependency names, or legal upstream references. They are not PastureStack product identity.

The neutral executable is `webhook-automation-service`. The Server release may install an internal `webhook-service` compatibility link until the preserved launcher property is changed. Public release assets use only the neutral name.

Compatibility is deliberately constrained where old behavior conflicts with security:

- The service never reads `RSA_PRIVATE_KEY_CONTENTS` or a private-key file. RS256 verification requires only the public key.
- The listener defaults to `127.0.0.1:8085`; a different address requires explicit `PASTURESTACK_WEBHOOK_LISTEN_ADDRESS` configuration.
- Webhook bodies and forwarded responses are bounded. Oversized requests are rejected.
- Credentialed forward requests refuse redirects, bypass ambient HTTP proxies, and do not copy inbound authorization or cookie headers.
- Forward destinations are derived from the configured control-plane API authority and validated driver fields. The exact normalized scheme, host, and port are checked again immediately before network transport, so webhook input cannot select an arbitrary host or override the HTTP `Host` value.
- Existing tokens without time claims remain valid. If `exp` or `nbf` is present, it must be an integer NumericDate and is enforced.

Before release, validate all four drivers against the exact Server API, launcher supervision, Web Console create/list/delete flows, restart behavior, master-node failover, upgrade, rollback, and the immutable release archive in an isolated VM.
