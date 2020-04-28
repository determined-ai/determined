package internal

import (
	"encoding/json"
	"io/ioutil"
	"os"

	hclient "github.com/docker/docker-credential-helpers/client"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
)

const (
	dockerConfigFile        = "/root/.docker/config.json"
	credentialsHelperPrefix = "docker-credential-"
	tokenUsername           = "<token>"
)

type credentialStore struct {
	registry string
	store    hclient.ProgramFunc
}

// getAllCredentialStores returns the credential helpers configured in the default docker
// config or an error.
func getAllCredentialStores() (map[string]*credentialStore, error) {
	type ConfigFile struct {
		CredentialHelpers map[string]string `json:"credHelpers,omitempty"`
	}

	credentialsStores := map[string]*credentialStore{}
	configFile, err := os.Open(dockerConfigFile)
	if err != nil {
		return credentialsStores, errors.Wrap(err, "can't open docker config")
	}

	b, err := ioutil.ReadAll(configFile)
	if err != nil {
		return credentialsStores, errors.Wrap(err, "can't read docker config")
	}

	var config ConfigFile
	err = json.Unmarshal(b, &config)
	if err != nil {
		return credentialsStores, errors.Wrap(err, "can't parse docker config")
	}

	if config.CredentialHelpers == nil {
		return credentialsStores, nil
	}

	for hostname, helper := range config.CredentialHelpers {
		credentialsStores[hostname] = &credentialStore{
			registry: hostname,
			store:    hclient.NewShellProgramFunc(credentialsHelperPrefix + helper),
		}
	}

	return credentialsStores, nil
}

// get executes the command to get the credentials from the native store.
func (s *credentialStore) get() (types.AuthConfig, error) {
	var ret types.AuthConfig

	creds, err := hclient.Get(s.store, s.registry)
	if err != nil {
		return ret, err
	}

	if creds.Username == tokenUsername {
		ret.IdentityToken = creds.Secret
	} else {
		ret.Password = creds.Secret
		ret.Username = creds.Username
	}

	ret.ServerAddress = s.registry
	return ret, nil
}
