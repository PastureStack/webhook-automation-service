module github.com/PastureStack/webhook-automation-service

go 1.26.0

toolchain go1.27.0

require (
	github.com/go-viper/mapstructure/v2 v2.5.0
	github.com/gorilla/mux v1.8.1
	github.com/rancher/go-rancher v0.1.1-0.20190109212254-cbc1b0a3f68d
	github.com/sirupsen/logrus v1.10.1
	github.com/urfave/cli/v3 v3.11.0
)

require (
	github.com/gorilla/context v1.1.2 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace github.com/rancher/go-rancher => ./third_party/go-rancher
