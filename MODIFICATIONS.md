# PastureStack Modifications

The consolidated PastureStack maintenance change:

- gives the repository and executable the semantic `webhook-automation-service` name;
- preserves the v1.6 webhook routes and four driver identifiers required by existing clients;
- replaces private-key loading and the retired JWT library with public-key-only standard-library RS256 verification;
- removes committed test keys and generates ephemeral test keys instead;
- limits HTTP bodies and headers, adds timeouts, binds credentialed forwarding to the exact configured origin, disables ambient proxies, refuses redirects, and filters sensitive forwarded headers;
- fixes no-query forwarding, unsafe URL slicing, unchecked nested payload assertions, selector matching, and response-body handling;
- adds neutral environment names while retaining documented launcher aliases;
- replaces the retired Dapper/Drone build with direct Go 1.26 validation, race, integration, content-policy, deterministic packaging, artifact verification, CodeQL verification, and a short-lived security-evidence workflow; and
- includes release-source and composite license material without changing the inherited Apache-2.0 license.

Release publication and deployment remain disabled until the exact source and archive complete the isolated-VM acceptance gates.
