module github.com/determined-ai/determined/master

go 1.15

require (
	cloud.google.com/go v0.58.0
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.9 // indirect
	github.com/aws/aws-sdk-go v1.34.32
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bufbuild/buf v0.16.0
	github.com/containerd/containerd v1.3.2 // indirect
	github.com/determined-ai/determined/proto v0.0.0-00010101000000-000000000000
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0 // indirect
	github.com/dustinkirkland/golang-petname v0.0.0-20170921220637-d3c2ba80e75e
	github.com/elastic/go-elasticsearch/v7 v7.9.0
	github.com/emirpasic/gods v1.12.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/golang-migrate/migrate v3.5.4+incompatible
	github.com/golang/protobuf v1.4.2
	github.com/golangci/golangci-lint v1.28.3
	github.com/google/go-cmp v0.4.1
	github.com/google/uuid v1.1.1
	github.com/goreleaser/goreleaser v0.140.0
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/grpc-ecosystem/grpc-gateway v1.14.6
	github.com/jmoiron/sqlx v1.2.1-0.20190826204134-d7d95172beb5
	github.com/labstack/echo v3.3.5+incompatible
	github.com/labstack/gommon v0.0.0-20180613044413-d6898124de91
	github.com/lib/pq v1.8.0
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/o1egl/paseto v1.0.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/segmentio/backo-go v0.0.0-20200129164019-23eae7c10bd3 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/soheilhy/cmux v0.1.4
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	github.com/uber/jaeger-client-go v2.25.0+incompatible
	github.com/uber/jaeger-lib v2.2.0+incompatible // indirect
	github.com/valyala/fasttemplate v0.0.0-20170224212429-dcecefd839c4 // indirect
	github.com/xtgo/uuid v0.0.0-20140804021211-a0b114877d4c // indirect
	golang.org/x/crypto v0.0.0-20200510223506-06a226fb4e37
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9
	golang.org/x/tools v0.0.0-20200702044944-0cc1aa72b347
	google.golang.org/api v0.26.0
	google.golang.org/genproto v0.0.0-20200608115520-7c474a2e3482
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.24.0
	gopkg.in/guregu/null.v3 v3.4.0
	gopkg.in/segmentio/analytics-go.v3 v3.1.0
	gotest.tools v2.1.0+incompatible
	k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb
	k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
)

replace github.com/determined-ai/determined/proto => ../proto

replace github.com/docker/docker v1.13.1 => github.com/docker/engine v1.4.2-0.20191113042239-ea84732a7725
