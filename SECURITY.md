# Security Policy

## Supported migration target

Security review covers the single PastureStack maintenance change built from the preserved upstream `v0.9.15` boundary. It remains a migration candidate until the exact archive passes the isolated-VM gates in [COMPATIBILITY.md](COMPATIBILITY.md).

## Runtime requirements

- Keep the service on loopback unless a separately reviewed network and access-control boundary requires another address.
- Supply API credentials and the RSA public key through the launcher environment or a protected file, never source control or command-line history.
- Do not provide the control-plane private key. It is unnecessary and intentionally ignored.
- Protect receiver URLs and JWTs as credentials. Revoke a receiver immediately if its URL or token is disclosed.
- Keep the control-plane API URL on a trusted internal endpoint. It must be an unambiguous HTTP or HTTPS URL without embedded credentials, a query, or a fragment. Credentialed redirects and ambient HTTP proxies are refused.
- Verify the GitHub Release archive and executable SHA-256 digests before installation.
- Do not place active credentials, private addresses, payloads, or environment exports in issues or logs.

The service enforces exact RS256 signatures, caps token and body sizes, validates optional time claims, applies HTTP timeouts, limits server headers, filters sensitive forwarded headers, revalidates the exact destination origin at the transport boundary, and avoids logging webhook bodies or API credentials. These controls reduce risk but do not make a webhook receiver safe to expose without authentication and network policy.

## Reporting

Use GitHub private vulnerability reporting for security findings. Provide a minimal synthetic reproducer and omit production secrets or private infrastructure details.
