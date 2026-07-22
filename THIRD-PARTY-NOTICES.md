# Third-Party Notices

The executable is built from the exact vendored source in this repository. The vendored runtime dependencies are:

- Gorilla context, mux, and WebSocket;
- Mitchell Hashimoto's mapstructure;
- `pkg/errors`;
- the inherited `go-rancher` API client;
- Logrus;
- urfave/cli; and
- the Go standard library.

The vendored `gopkg.in/check.v1` package is used by tests and is not linked into the runtime executable. License and author files remain beside the vendored source. The deterministic release archive concatenates the project license, every tracked vendored `LICENSE`, `COPYING`, `NOTICE`, and `AUTHORS` file, and the Go standard-library license into `webhook-automation-service-LICENSES.txt`.

Exact inherited revisions for the retained vendored source are recorded in [`vendor.lock`](vendor.lock). The lock is provenance for the preserved source tree, not a claim that those historical revisions are current releases.

The retired `dgrijalva/jwt-go`, `rancher-auth-service`, and `dchest/uniuri` code is not present in the current tree or build graph. Their historical source remains attributable at the unchanged upstream commits.

- Preserved upstream source: `https://github.com/rancher/webhook-service`
- PastureStack source: `https://github.com/PastureStack/webhook-automation-service`
- Go source and license: `https://go.dev/`

Copyright and license terms remain with their respective authors. PastureStack does not claim ownership of inherited or vendored work.
