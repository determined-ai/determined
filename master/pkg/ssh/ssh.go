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
	rsaPEMBlockType   = "RSA PRIVATE KEY"
	ecdsaPEMBlockType = "EC PRIVATE KEY"
)

// PrivateAndPublicKeys contains a private and public key.
type PrivateAndPublicKeys struct {
	PrivateKey []byte
	PublicKey  []byte
}

// GenerateKey returns a private and public SSH key.
func GenerateKey(conf config.SSHConfig) (PrivateAndPublicKeys, error) {
	var generatedKeys PrivateAndPublicKeys
	switch conf.KeyType {
	case config.KeyTypeRSA:
		return generateRSAKey(conf.RsaKeySize)
	case config.KeyTypeECDSA:
		return generateECDSAKey()
	case config.KeyTypeED25519:
		return generateED25519Key()
	default:
		return generatedKeys, errors.New("Invalid crypto system")
	}
}

func generateRSAKey(rsaKeySize int) (PrivateAndPublicKeys, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		return PrivateAndPublicKeys{}, errors.Wrap(err, "unable to generate RSA private key")
	}

	if err = privateKey.Validate(); err != nil {
		return PrivateAndPublicKeys{}, err
	}

	block := &pem.Block{
		Type:  rsaPEMBlockType,
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	publicKey, err := sshlib.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return PrivateAndPublicKeys{}, errors.Wrap(err, "unable to generate RSA public key")
	}

	return PrivateAndPublicKeys{
		PrivateKey: pem.EncodeToMemory(block),
		PublicKey:  sshlib.MarshalAuthorizedKey(publicKey),
	}, nil
}

func generateECDSAKey() (PrivateAndPublicKeys, error) {
	// Curve size currently not configurable, using the NIST recommendation.
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return PrivateAndPublicKeys{}, errors.Wrap(err, "unable to generate ECDSA private key")
	}

	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return PrivateAndPublicKeys{}, errors.Wrap(err, "unable to marshal ECDSA private key")
	}

	block := &pem.Block{
		Type:  ecdsaPEMBlockType,
		Bytes: privateKeyBytes,
	}

	publicKey, err := sshlib.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return PrivateAndPublicKeys{}, errors.Wrap(err, "unable to generate ECDSA public key")
	}

	return PrivateAndPublicKeys{
		PrivateKey: pem.EncodeToMemory(block),
		PublicKey:  sshlib.MarshalAuthorizedKey(publicKey),
	}, nil
}

func generateED25519Key() (PrivateAndPublicKeys, error) {

	ed25519PublicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return PrivateAndPublicKeys{}, errors.Wrap(err, "unable to generate ED25519 private key")
	}

	// Before OpenSSH 9.6, for ED25519 keys, only the OpenSSH private key format was supported.
	block, err := sshlib.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return PrivateAndPublicKeys{}, errors.Wrap(err, "unable to marshal ED25519 private key")
	}

	publicKey, err := sshlib.NewPublicKey(ed25519PublicKey)
	if err != nil {
		return PrivateAndPublicKeys{}, errors.Wrap(err, "unable to generate ED25519 public key")
	}

	return PrivateAndPublicKeys{
		PrivateKey: pem.EncodeToMemory(block),
		PublicKey:  sshlib.MarshalAuthorizedKey(publicKey),
	}, nil

}
