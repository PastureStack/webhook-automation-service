# Third-Party Notices

The executable is built from the exact Go Module graph and vendor tree in this repository. The runtime dependencies are:

- Gorilla context, mux, and WebSocket;
- go-viper mapstructure v2;
- the reviewed Rancher v1 `go-rancher` compatibility subset in `third_party/go-rancher`;
- Logrus 1.10;
- urfave/cli v3;
- `golang.org/x/sys`; and
- the Go standard library.

Tests use the Go standard testing package. License and author files remain beside vendored source. The deterministic release archive concatenates the project license, every tracked vendored `LICENSE`, `COPYING`, `NOTICE`, and `AUTHORS` file, and the Go standard-library license into `webhook-automation-service-LICENSES.txt`.

`go.mod`, `go.sum`, and `vendor/modules.txt` are the dependency lock. The local compatibility module is derived from upstream `go-rancher` commit `cbc1b0a3f68db14c8d074ffc7fe2ca3b26a82c91`; its scope is limited to the `api`, `client`, and `v2` packages required by the preserved wire contract.

The retired `dgrijalva/jwt-go`, `rancher-auth-service`, and `dchest/uniuri` code is not present in the current tree or build graph. Their historical source remains attributable at the unchanged upstream commits.

- Preserved upstream source: `https://github.com/rancher/webhook-service`
- PastureStack source: `https://github.com/PastureStack/webhook-automation-service`
- Go source and license: `https://go.dev/`

Copyright and license terms remain with their respective authors. PastureStack does not claim ownership of inherited or vendored work.
