module github.com/determined-ai/determined/agent

go 1.12

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/determined-ai/determined/master v0.0.0
	github.com/determined-ai/determined/proto v0.0.0-00010101000000-000000000000
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/docker/docker-credential-helpers v0.6.3
	github.com/docker/go-connections v0.4.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golangci/golangci-lint v1.28.3
	github.com/google/uuid v1.1.1
	github.com/goreleaser/goreleaser v0.140.0
	github.com/gorilla/websocket v1.4.2
	github.com/labstack/echo v3.3.5+incompatible
	github.com/pkg/errors v0.9.1
	github.com/shirou/gopsutil v2.19.9+incompatible
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/sys v0.0.0-20200602225109-6fdc65e7d980
	golang.org/x/tools v0.0.0-20200702044944-0cc1aa72b347
	gotest.tools v2.2.0+incompatible
)

replace github.com/determined-ai/determined/master => ../master

replace github.com/determined-ai/determined/proto => ../proto

replace github.com/docker/docker v1.13.1 => github.com/docker/engine v1.4.2-0.20191113042239-ea84732a7725
