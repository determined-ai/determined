package elastic

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

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
	cfg := elasticsearch.Config{
		Addresses: []string{fmt.Sprintf("%s%s:%d", scheme, conf.Host, conf.Port)},
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
	return &Elastic{es}, nil
}

func elasticTLSConfig(conf model.ElasticTLSConfig) (*tls.Config, error) {
	if !conf.Enabled {
		return nil, nil
	}

	var pool *x509.CertPool
	if certFile := conf.CertificatePath; certFile != "" {
		certData, err := ioutil.ReadFile(certFile) //nolint:gosec
		if err != nil {
			return nil, errors.Wrap(err, "failed to read certificate file")
		}
		pool = x509.NewCertPool()
		if !pool.AppendCertsFromPEM(certData) {
			return nil, errors.New("certificate file contains no certificates")
		}
	}

	return &tls.Config{
		InsecureSkipVerify: conf.SkipVerify, //nolint:gosec
		MinVersion:         tls.VersionTLS11,
		RootCAs:            pool,
	}, nil
}
