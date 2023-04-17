package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	sshlib "golang.org/x/crypto/ssh"
)

const (
	trialPEMBlockType = "RSA PRIVATE KEY"
)

// PrivateAndPublicKeys contains a private and public key.
type PrivateAndPublicKeys struct {
	PrivateKey []byte
	PublicKey  []byte
}

// GenerateKey returns a private and public SSH key.
func GenerateKey(rsaKeySize int, passphrase *string) (PrivateAndPublicKeys, error) {
	var generatedKeys PrivateAndPublicKeys
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		return generatedKeys, fmt.Errorf("unable to generate private key: %w", err)
	}

	if err = privateKey.Validate(); err != nil {
		return generatedKeys, err
	}

	block := &pem.Block{
		Type:  trialPEMBlockType,
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	if passphrase != nil {
		// TODO: Replace usage of deprecated x509.EncryptPEMBlock.
		block, err = x509.EncryptPEMBlock( //nolint: staticcheck
			rand.Reader, block.Type, block.Bytes, []byte(*passphrase), x509.PEMCipherAES256)
		if err != nil {
			return generatedKeys, fmt.Errorf("unable to encrypt private key: %w", err)
		}
	}

	publicKey, err := sshlib.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return generatedKeys, fmt.Errorf("unable to generate public key: %w", err)
	}

	generatedKeys = PrivateAndPublicKeys{
		PrivateKey: pem.EncodeToMemory(block),
		PublicKey:  sshlib.MarshalAuthorizedKey(publicKey),
	}

	return generatedKeys, nil
}
