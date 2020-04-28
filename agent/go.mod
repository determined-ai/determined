module github.com/determined-ai/determined/agent

go 1.12

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/containerd/continuity v0.0.0-20200413184840-d3ef23f19fbb // indirect
	github.com/containerd/fifo v0.0.0-20200410184934-f15a3290365b // indirect
	github.com/containerd/ttrpc v1.0.0 // indirect
	github.com/containerd/typeurl v1.0.0 // indirect
	github.com/determined-ai/determined/master v0.0.0
	github.com/docker/cli v0.0.0-20200303215952-eb310fca4956
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/docker/docker-credential-helpers v0.6.3 // indirect
	github.com/docker/go v1.5.1-1 // indirect
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/gogo/googleapis v1.4.0 // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/google/uuid v1.0.0
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/labstack/echo v3.3.5+incompatible
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/opencontainers/runtime-spec v1.0.2 // indirect
	github.com/pkg/errors v0.8.1
	github.com/shirou/gopsutil v2.19.9+incompatible
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/syndtr/gocapability v0.0.0-20180916011248-d98352740cb2 // indirect
	github.com/theupdateframework/notary v0.6.1 // indirect
	golang.org/x/sys v0.0.0-20191210023423-ac6580df4449
	gotest.tools v2.2.0+incompatible
	vbom.ml/util v0.0.0-20180919145318-efcd4e0f9787 // indirect
)

replace github.com/determined-ai/determined/master => ../master

replace github.com/docker/docker v1.13.1 => github.com/docker/engine v1.4.2-0.20191113042239-ea84732a7725
