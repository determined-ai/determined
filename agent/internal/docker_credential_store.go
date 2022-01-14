package internal

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	hclient "github.com/docker/docker-credential-helpers/client"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
)

const (
	//nolint:gosec // It thinks these are credentials...
	credentialsHelperPrefix = "docker-credential-"
	tokenUsername           = "<token>"
)

type credentialStore struct {
	registry string
	store    hclient.ProgramFunc
}

func getDockerConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return homeDir, errors.Wrap(err, "unable to find user's HOME directory")
	}
	return path.Join(homeDir, ".docker", "config.json"), nil
}

// getAllCredentialStores returns the credential helpers configured in the default docker
// config or an error.
func getAllCredentialStores() (map[string]*credentialStore, error) {
	type ConfigFile struct {
		CredentialHelpers map[string]string `json:"credHelpers,omitempty"`
	}

	credentialsStores := map[string]*credentialStore{}
	dockerConfigFile, err := getDockerConfigPath()
	if err != nil {
		return credentialsStores, err
	}

	configFile, err := os.Open(dockerConfigFile) // #nosec: G304
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
