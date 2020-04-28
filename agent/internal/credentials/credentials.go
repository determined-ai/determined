package credentials

import (
	"sync"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
)

var docker *configfile.ConfigFile
var lock sync.Mutex

func init() {
	docker = config.LoadDefaultConfigFile(nil)
}

func Get(ref reference.Named, auth *types.AuthConfig) (string, error) {
	if auth == nil {
		repoInfo, err := registry.ParseRepositoryInfo(ref)
		if err != nil {
			return "", errors.Wrapf(err, "error parsing repository info: %s", ref)
		}
		lock.Lock()
		defer lock.Unlock()
		a, err := docker.GetAuthConfig(repoInfo.Index.Name)
		if err != nil {
			return "", errors.Wrapf(err, "error getting auth config: %s", repoInfo.Index.Name)
		}
		return command.EncodeAuthToBase64(types.AuthConfig(a))
	}
	return command.EncodeAuthToBase64(*auth)
}
