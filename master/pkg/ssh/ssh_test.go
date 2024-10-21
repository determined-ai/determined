package ssh

import (
	"testing"

	"golang.org/x/crypto/ssh"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/config"
)

func verifyKeys(t *testing.T, keys PrivateAndPublicKeys) {
	privateKey, err := ssh.ParsePrivateKey(keys.PrivateKey)
	assert.NilError(t, err)

	publickKey, _, _, _, err := ssh.ParseAuthorizedKey(keys.PublicKey) //nolint:dogsled
	assert.NilError(t, err)
	assert.Equal(t, string(publickKey.Marshal()), string(privateKey.PublicKey().Marshal()))
}

func TestSSHKeyGenerate(t *testing.T) {
	keys, err := GenerateKey(config.SSHConfig{KeyType: config.RSACryptoSystem, RsaKeySize: 1024})
	assert.NilError(t, err)
	verifyKeys(t, keys)

	keys, err = GenerateKey(config.SSHConfig{KeyType: config.ECDSACryptoSystem})
	assert.NilError(t, err)
	verifyKeys(t, keys)

	keys, err = GenerateKey(config.SSHConfig{KeyType: config.ED25519CryptoSystem})
	assert.NilError(t, err)
	verifyKeys(t, keys)
}
