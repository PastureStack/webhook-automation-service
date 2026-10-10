# Compatibility Contract

The migration preserves the v1.6 surface consumed by the orchestration engine and Web Console:

- `GET /v1-webhooks` and the inherited schemas, receivers, and endpoint routes;
- the `scaleService`, `scaleHost`, `serviceUpgrade`, and `forwardPost` driver identifiers;
- loopback port `8085`;
- `RSA_PUBLIC_KEY_CONTENTS` as a secondary alias for `PASTURESTACK_API_PUBLIC_KEY_CONTENTS`;
- `CATTLE_URL`, `CATTLE_ACCESS_KEY`, and `CATTLE_SECRET_KEY` as secondary aliases for `PASTURESTACK_API_URL`, `PASTURESTACK_API_ACCESS_KEY`, and `PASTURESTACK_API_SECRET_KEY`; and
- the inherited `go-rancher` API types, label keys, placeholder image value, and `/v2-beta` API behavior needed by the preserved wire contract.

The legacy product strings above are compatibility identifiers, dependency names, or legal upstream references. They are not PastureStack product identity.

The neutral executable is `webhook-automation-service`. Server images retain an internal `webhook-service` compatibility link for the preserved launcher property. Public release assets use only the neutral name.

Compatibility is deliberately constrained where old behavior conflicts with security:

- The service never reads `RSA_PRIVATE_KEY_CONTENTS` or a private-key file. RS256 verification requires only the public key.
- The listener defaults to `127.0.0.1:8085`; a different address requires explicit `PASTURESTACK_WEBHOOK_LISTEN_ADDRESS` configuration.
- Webhook bodies and forwarded responses are bounded. Oversized requests are rejected.
- Credentialed forward requests refuse redirects, bypass ambient HTTP proxies, and do not copy inbound authorization or cookie headers.
- Forward destinations are derived from the configured control-plane API authority and validated driver fields. The exact normalized scheme, host, and port are checked again immediately before network transport, so webhook input cannot select an arbitrary host or override the HTTP `Host` value.
- Existing tokens without time claims remain valid. If `exp` or `nbf` is present, it must be an integer NumericDate and is enforced.

The published v0.10.3 artifact passed its Linux component security release gate. Its Receiver schema now advertises the existing project-role write boundary, and role-specific responses are non-cacheable. Before claiming Server 8080 acceptance for this patch, validate the bundled component and Web Console against the same owner/member/restricted/readonly/no-access Receiver create/list/delete and direct-route flows, then verify restart and rollback in an isolated environment. Other driver or failover acceptance remains separate; this component release alone does not establish it.
