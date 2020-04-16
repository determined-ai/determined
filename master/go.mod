module github.com/determined-ai/determined/master

require (
	cloud.google.com/go v0.38.0
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.9 // indirect
	github.com/aws/aws-sdk-go v1.20.2
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/containerd/containerd v1.3.2 // indirect
	github.com/d4l3k/go-bayesopt v0.0.0-20191110222447-8506d3040732
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0 // indirect
	github.com/dustinkirkland/golang-petname v0.0.0-20170921220637-d3c2ba80e75e
	github.com/emirpasic/gods v1.12.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-sql-driver/mysql v1.4.0 // indirect
	github.com/golang-migrate/migrate v3.5.4+incompatible
	github.com/google/go-cmp v0.3.0
	github.com/google/uuid v1.0.0
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/gorilla/websocket v1.4.0
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jmoiron/sqlx v0.0.0-20180614180643-0dae4fefe7c0
	github.com/labstack/echo v3.3.5+incompatible
	github.com/labstack/gommon v0.0.0-20180613044413-d6898124de91
	github.com/lib/pq v1.0.0
	github.com/mattn/go-colorable v0.0.9 // indirect
	github.com/mattn/go-isatty v0.0.4 // indirect
	github.com/mattn/go-sqlite3 v1.9.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/o1egl/paseto v1.0.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pkg/errors v0.8.1
	github.com/segmentio/backo-go v0.0.0-20200129164019-23eae7c10bd3 // indirect
	github.com/sirupsen/logrus v1.4.1
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.3.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v0.0.0-20170224212429-dcecefd839c4 // indirect
	github.com/xtgo/uuid v0.0.0-20140804021211-a0b114877d4c // indirect
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	golang.org/x/net v0.0.0-20190522155817-f3200d17e092
	google.golang.org/api v0.7.0
	gopkg.in/guregu/null.v3 v3.4.0
	gopkg.in/segmentio/analytics-go.v3 v3.1.0
	gotest.tools v2.1.0+incompatible
)

replace github.com/docker/docker v1.13.1 => github.com/docker/engine v1.4.2-0.20191113042239-ea84732a7725

go 1.13
