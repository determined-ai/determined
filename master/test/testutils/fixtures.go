// +build integration

package testutils

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/determined-ai/determined/master/internal/elastic"
	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/sirupsen/logrus"

	"github.com/ghodss/yaml"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/determined-ai/determined/master/internal"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/version"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	defaultUsername     = "determined"
	defaultMasterConfig = `
checkpoint_storage:
  type: shared_fs
  host_path: /tmp

db:
  user: postgres
  password: postgres
  name: determined
  migrations: file://../../../static/migrations

root: ../../..
`
)

// ResolvePostgres resolves a connection to a postgres database.
func ResolvePostgres() (*db.PgDB, error) {
	pgDB, err := db.ConnectPostgres(os.Getenv("DET_INTEGRATION_POSTGRES_URL"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	return pgDB, nil
}

// ResolveElastic resolves a connection to an elasticsearch database.
func ResolveElastic() (*elastic.Elastic, error) {
	es, err := elastic.Setup(*DefaultElasticConfig().ElasticLoggingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to elasticsearch: %w", err)
	}
	return es, nil
}

// RunMaster runs a master in a goroutine and returns a reference to the master,
// along with all the external context required to interact with the master, and
// a function to close it.
func RunMaster(ctx context.Context, c *internal.Config) (
	*internal.Master, *logger.LogBuffer, apiv1.DeterminedClient,
	context.Context, error,
) {
	if c == nil {
		dConf, err := DefaultMasterConfig()
		if err != nil {
			return nil, nil, nil, nil, err
		}
		c = dConf
	}
	logs := logger.NewLogBuffer(100)
	m := internal.New(version.Version, logs, c)
	logrus.AddHook(logs)
	go func() {
		err := m.Run(ctx)
		switch {
		case err == context.Canceled:
			fmt.Println("master stopped")
		case err != nil:
			fmt.Println("error running master: ", err)
		}
	}()

	cl, err := ConnectMaster(c)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	creds, err := APICredentials(context.Background(), cl)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return m, logs, cl, creds, nil
}

// ConnectMaster blocks until a connection can be made to this master, assumed to be running
// on localhost on the port indicated by the configuration. Returns an error if unable to connect
// after 5 tries with 100ms delay between each.
func ConnectMaster(c *internal.Config) (apiv1.DeterminedClient, error) {
	var cl apiv1.DeterminedClient
	var clConn *grpc.ClientConn
	var err error
	for i := 0; i < 15; i++ {
		clConn, err = grpc.Dial(fmt.Sprintf("localhost:%d", c.Port), grpc.WithInsecure())
		if err != nil {
			err = fmt.Errorf("failed to dial master: %w", err)
			continue
		}

		cl = apiv1.NewDeterminedClient(clConn)
		_, err = cl.Login(context.Background(), &apiv1.LoginRequest{Username: defaultUsername})
		if err == nil {
			return cl, nil
		}
		time.Sleep(time.Second)
	}
	return nil, fmt.Errorf("failed to connect to master: %w", err)
}

// DefaultMasterConfig returns the default master configuration.
func DefaultMasterConfig() (*internal.Config, error) {
	c := internal.DefaultConfig()
	if err := yaml.Unmarshal([]byte(defaultMasterConfig), c, yaml.DisallowUnknownFields); err != nil {
		return nil, err
	}

	if err := c.Resolve(); err != nil {
		return nil, err
	}

	if err := check.Validate(c); err != nil {
		return nil, err
	}
	return c, nil
}

func DefaultElasticConfig() model.LoggingConfig {
	port, err := strconv.Atoi(os.Getenv("DET_INTEGRATION_ES_PORT"))
	if err != nil {
		panic("elastic config had non-numeric port")
	}
	return model.LoggingConfig{
		ElasticLoggingConfig: &model.ElasticLoggingConfig{
			Host: os.Getenv("DET_INTEGRATION_ES_HOST"),
			Port: port,
		},
	}
}

// APICredentials takes a context and a connected apiv1.DeterminedClient and returns a context
// with credentials or an error if unable to login with defaults.
func APICredentials(ctx context.Context, cl apiv1.DeterminedClient) (context.Context, error) {
	resp, err := cl.Login(context.TODO(), &apiv1.LoginRequest{Username: defaultUsername})
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	return metadata.AppendToOutgoingContext(
		ctx, "x-user-token", fmt.Sprintf("Bearer %s", resp.Token)), nil
}
