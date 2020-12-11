package elastic

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// Elastic is an interface around an elasticsearch client that abstracts away
// common queries and indexing operations.
type Elastic struct {
	client *elasticsearch.Client
}

// Setup sets up a new elasticsearch client with the given configuration.
func Setup(conf model.ElasticLoggingConfig) (*Elastic, error) {
	tlsCfg, err := elasticTLSConfig(conf.Security.TLS)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make elastic tls config")
	}

	var scheme string
	if tlsCfg != nil {
		scheme = "https://"
	} else {
		scheme = "http://"
	}
	addr := fmt.Sprintf("%s%s:%d", scheme, conf.Host, conf.Port)
	logrus.Infof("connecting to elasticsearch %s", addr)

	cfg := elasticsearch.Config{
		Addresses: []string{addr},
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Second,
			DialContext:           (&net.Dialer{Timeout: time.Second}).DialContext,
			TLSClientConfig:       tlsCfg,
		},
	}

	if conf.Security.Username != nil && conf.Security.Password != nil {
		cfg.Username = *conf.Security.Username
		cfg.Password = *conf.Security.Password
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create elastic client from config")
	}

	// Try to connect to elastic - we'd rather fail hard here than on first log write.
	numTries := 0
	for {
		i, err := es.Info()
		if err == nil {
			logrus.Infof("connected to elasticsearch cluster with info: %s", i.String())
			return &Elastic{es}, nil
		}
		numTries++
		if numTries >= 15 {
			return nil, errors.Wrapf(err, "could not connect to elastic after %v tries", numTries)
		}
		time.Sleep(time.Second)
	}
}

func elasticTLSConfig(conf model.ElasticTLSConfig) (*tls.Config, error) {
	if !conf.Enabled {
		return nil, nil
	}

	var pool *x509.CertPool
	if conf.CertBytes != nil {
		pool = x509.NewCertPool()
		if !pool.AppendCertsFromPEM(conf.CertBytes) {
			return nil, errors.New("certificate file contains no certificates")
		}
	}

	return &tls.Config{
		InsecureSkipVerify: conf.SkipVerify, //nolint:gosec
		MinVersion:         tls.VersionTLS12,
		RootCAs:            pool,
	}, nil
}
