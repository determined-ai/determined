package ssh

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/pkg/errors"
	sshlib "golang.org/x/crypto/ssh"

	"github.com/determined-ai/determined/master/internal/config"
)

const (
	rsaPEMBlockType     = "RSA PRIVATE KEY"
	ecdsaPEMBlockType   = "ECDSA PRIVATE KEY"
	ed25519PEMBlockType = "ED25519 PRIVATE KEY"
)

// PrivateAndPublicKeys contains a private and public key.
type PrivateAndPublicKeys struct {
	PrivateKey []byte
	PublicKey  []byte
}

// GenerateKey returns a private and public SSH key.
func GenerateKey(conf config.SSHConfig, passphrase *string) (PrivateAndPublicKeys, error) {
	var generatedKeys PrivateAndPublicKeys
	switch conf.CryptoSystem {
	case config.RSACryptoSystem:
		return generateRSAKey(conf.RsaKeySize, passphrase)
	case config.ECDSACryptoSystem:
		return generateECDSAKey(passphrase)
	case config.ED25519CryptoSystem:
		return generateED25519Key(passphrase)
	default:
		return generatedKeys, errors.New("Invalid crypto system")
	}
}

func generateRSAKey(rsaKeySize int, passphrase *string) (PrivateAndPublicKeys, error) {
	var generatedKeys PrivateAndPublicKeys
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		return generatedKeys, errors.Wrap(err, "unable to generate RSA private key")
	}

	if err = privateKey.Validate(); err != nil {
		return generatedKeys, err
	}

	block := &pem.Block{
		Type:  rsaPEMBlockType,
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	block, err = encodePassphrase(block, passphrase)
	if err != nil {
		return generatedKeys, errors.Wrap(err, "unable to encrypt RSA private key")
	}

	publicKey, err := sshlib.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return generatedKeys, errors.Wrap(err, "unable to generate RSA public key")
	}

	generatedKeys = PrivateAndPublicKeys{
		PrivateKey: pem.EncodeToMemory(block),
		PublicKey:  sshlib.MarshalAuthorizedKey(publicKey),
	}

	return generatedKeys, nil
}

func generateECDSAKey(passphrase *string) (PrivateAndPublicKeys, error) {
	var generatedKeys PrivateAndPublicKeys
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return generatedKeys, errors.Wrap(err, "unable to generate ECDS private key")
	}

	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return generatedKeys, errors.Wrap(err, "unable to marshal ECDS private key")
	}

	block := &pem.Block{
		Type:  ecdsaPEMBlockType,
		Bytes: privateKeyBytes,
	}

	block, err = encodePassphrase(block, passphrase)
	if err != nil {
		return generatedKeys, errors.Wrap(err, "unable to encrypt ECDS private key")
	}

	publicKey, err := sshlib.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return generatedKeys, errors.Wrap(err, "unable to generate ECDS public key")
	}

	generatedKeys = PrivateAndPublicKeys{
		PrivateKey: pem.EncodeToMemory(block),
		PublicKey:  sshlib.MarshalAuthorizedKey(publicKey),
	}

	return generatedKeys, nil
}

func generateED25519Key(passphrase *string) (PrivateAndPublicKeys, error) {
	var generatedKeys PrivateAndPublicKeys

	ed25519PublicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return generatedKeys, errors.Wrap(err, "unable to generate ED25519 private key")
	}
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return generatedKeys, errors.Wrap(err, "unable to marshal ED25519 private key")
	}

	block := &pem.Block{
		Type:  ed25519PEMBlockType,
		Bytes: privateKeyBytes,
	}

	block, err = encodePassphrase(block, passphrase)
	if err != nil {
		return generatedKeys, errors.Wrap(err, "unable to encrypt ED25519 private key")
	}

	publicKey, err := sshlib.NewPublicKey(ed25519PublicKey)
	if err != nil {
		return generatedKeys, errors.Wrap(err, "unable to generate ECDS public key")
	}

	generatedKeys = PrivateAndPublicKeys{
		PrivateKey: pem.EncodeToMemory(block),
		PublicKey:  sshlib.MarshalAuthorizedKey(publicKey),
	}

	return generatedKeys, nil
}

func encodePassphrase(block *pem.Block, passphrase *string) (*pem.Block, error) {
	if passphrase != nil {
		// TODO: Replace usage of deprecated x509.EncryptPEMBlock.
		return x509.EncryptPEMBlock( //nolint: staticcheck
			rand.Reader, block.Type, block.Bytes, []byte(*passphrase), x509.PEMCipherAES256)
	}
	return block, nil
}
