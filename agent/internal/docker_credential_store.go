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

// processDockerConfig reads a users ~/.docker/config.json and returns
// credential helpers configured and the "auths" section of the config.
func processDockerConfig() (map[string]*credentialStore, map[string]types.AuthConfig, error) {
	credStores := make(map[string]*credentialStore) // Always return non nil maps.
	authConfig := make(map[string]types.AuthConfig)

	dockerConfigFile, err := getDockerConfigPath()
	if err != nil {
		return credStores, authConfig, err
	}
	configFile, err := os.Open(dockerConfigFile) // #nosec: G304
	if err != nil {
		return credStores, authConfig, errors.Wrap(err, "can't open docker config")
	}
	b, err := ioutil.ReadAll(configFile)
	if err != nil {
		return credStores, authConfig, errors.Wrap(err, "can't read docker config")
	}

	var config struct {
		CredentialHelpers map[string]string           `json:"credHelpers"`
		Auths             map[string]types.AuthConfig `json:"auths"`
	}
	if err := json.Unmarshal(b, &config); err != nil {
		return credStores, authConfig, errors.Wrap(err, "can't parse docker config")
	}
	for hostname, helper := range config.CredentialHelpers {
		credStores[hostname] = &credentialStore{
			registry: hostname,
			store:    hclient.NewShellProgramFunc(credentialsHelperPrefix + helper),
		}
	}
	if config.Auths != nil {
		authConfig = config.Auths
	}
	return credStores, authConfig, nil
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
