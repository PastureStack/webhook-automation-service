# PastureStack Modifications

The consolidated PastureStack maintenance change:

- gives the repository and executable the semantic `webhook-automation-service` name;
- preserves the v1.6 webhook routes and four driver identifiers required by existing clients;
- replaces private-key loading and the retired JWT library with public-key-only standard-library RS256 verification;
- removes committed test keys and generates ephemeral test keys instead;
- limits HTTP bodies and headers, adds timeouts, binds credentialed forwarding to the exact configured origin, disables ambient proxies, refuses redirects, and filters sensitive forwarded headers;
- fixes no-query forwarding, unsafe URL slicing, unchecked nested payload assertions, selector matching, and response-body handling;
- adds neutral environment names while retaining documented launcher aliases;
- replaces the retired Dapper/Drone and GOPATH builds with Go 1.27 Modules, vendored reproducibility, race and integration tests, content-policy checks, deterministic packaging, artifact verification, CodeQL verification, and a short-lived security-evidence workflow;
- upgrades the maintained direct dependencies, including mapstructure v2, Logrus 1.10, urfave/cli v3, Gorilla packages, and `x/sys`, while retaining only the reviewed Rancher v1 API compatibility subset as a local module; and
- includes release-source and composite license material without changing the inherited Apache-2.0 license.

Release publication and deployment remain disabled until the exact source and archive complete the isolated-VM acceptance gates.
