package proxy

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
)

const https = "https"

var (
	masterKey        *rsa.PrivateKey
	masterCert       *x509.Certificate
	masterCAKey      *rsa.PrivateKey
	masterCACert     *x509.Certificate
	masterCAKeyBytes []byte
	masterInfoMutex  sync.Mutex
)

type certAndKeyInfo struct {
	bun.BaseModel `bun:"table:cert_and_key_info"`
	Serial        int64  `bun:"serial_number"`
	Cert          []byte `bun:"cert"`
	Key           []byte `bun:"key"`
	IsMaster      bool   `bun:"is_master"`
	IsCA          bool   `bun:"is_ca"`
}

func isCertExpired(certificate *x509.Certificate) bool {
	if certificate == nil || time.Now().After(certificate.NotAfter) {
		return true
	}
	return false
}

func saveCACertAndKey() error {
	logrus.Info("Saving newly generated CA certificate and key")
	value := &certAndKeyInfo{
		Serial:   masterCACert.SerialNumber.Int64(),
		Cert:     masterCACert.Raw,
		Key:      masterCAKeyBytes,
		IsMaster: false,
		IsCA:     true,
	}

	_, err := db.Bun().NewInsert().Model(value).Exec(context.TODO())
	if err != nil {
		return errors.Wrap(err, "error inserting certificate and key")
	}
	return nil
}

func loadCACertAndKey() error {
	if masterCACert != nil && !isCertExpired(masterCACert) {
		return nil
	}

	var value certAndKeyInfo
	err := db.Bun().NewSelect().Model(&value).Where("is_ca = true").Scan(context.TODO())
	if err != nil {
		return err
	}

	certBytes := value.Cert
	keyBytes := value.Key

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return errors.Wrap(err, "error parsing certificate")
	}

	if isCertExpired(cert) {
		logrus.Info("Certificate expired!")
		_, err = db.Bun().NewDelete().Model(&value).Where("is_ca = true").Exec(context.TODO())
		if err != nil {
			return errors.Wrap(err, "error deleting expired CA certificate")
		}
		return nil
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBytes)
	if err != nil {
		return errors.Wrap(err, "error parsing key")
	}

	logrus.Info("Loaded CA certificate and key from database")

	masterCAKeyBytes = keyBytes
	masterCACert = cert
	masterCAKey = key

	return nil
}

func saveMasterCertAndKey() error {
	logrus.Info("Saving master certificate and key")
	value := &certAndKeyInfo{
		Serial:   masterCert.SerialNumber.Int64(),
		Cert:     masterCert.Raw,
		Key:      x509.MarshalPKCS1PrivateKey(masterKey),
		IsMaster: true,
		IsCA:     false,
	}

	_, err := db.Bun().NewInsert().Model(value).Exec(context.TODO())
	if err != nil {
		return errors.Wrap(err, "error writing master cert and key to db")
	}
	return nil
}

func loadMasterCertAndKey() error {
	if masterCert != nil && !isCertExpired(masterCert) {
		return nil
	}

	var value certAndKeyInfo
	err := db.Bun().NewSelect().Model(&value).
		Where("is_master = true").
		Scan(context.TODO())
	if err != nil {
		return err
	}

	certBytes := value.Cert
	keyBytes := value.Key

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return errors.Wrap(err, "error parsing certificate")
	}

	if isCertExpired(cert) {
		logrus.Info("Master certificate expired!")
		_, err = db.Bun().NewDelete().Model(&value).Where("is_master = true").Exec(context.TODO())
		if err != nil {
			return errors.Wrap(err, "error deleting expired master certificate")
		}
		return nil
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBytes)
	if err != nil {
		return errors.Wrap(err, "error parsing key")
	}

	logrus.Info("Loaded master certificate and key from database")

	masterCert = cert
	masterKey = key

	return nil
}

func genKeyAndSignCert(unsignedCert, caCert *x509.Certificate, caKey *rsa.PrivateKey,
) (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	certBytes, err := x509.CreateCertificate(
		rand.Reader, unsignedCert, caCert, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}

	parsedCert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, nil, err
	}

	return key, parsedCert, nil
}

func loadOrGenSignedMasterCert() error {
	masterInfoMutex.Lock()
	defer masterInfoMutex.Unlock()
	err := loadMasterCertAndKey()
	if err != nil {
		return err
	}
	if masterCert != nil {
		return nil
	}

	logrus.Info("Generating a new certificate and key for master")
	if masterCAKey == nil || masterCACert == nil {
		return errors.New("unable to generate signed cert; generate master key and cert first")
	}

	random, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt))
	if err != nil {
		return err
	}
	unsignedCert := &x509.Certificate{
		SerialNumber: random,
		Subject: pkix.Name{
			Organization:  []string{"Determined Master"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
		},
		IPAddresses: []net.IP{},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(10, 0, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	key, signedCert, err := genKeyAndSignCert(unsignedCert, masterCACert, masterCAKey)
	if err != nil {
		return err
	}

	masterKey = key
	masterCert = signedCert
	return saveMasterCertAndKey()
}

// LoadOrGenCA generates a new CA cert and keypair if it does not exist in DB.
func LoadOrGenCA() error {
	err := loadCACertAndKey()
	if err != nil {
		return err
	}
	if masterCACert != nil {
		return nil
	}

	logrus.Info("Generating a new CA certificate and key")
	random, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return err
	}
	caCert := &x509.Certificate{
		SerialNumber: random,
		Subject: pkix.Name{
			Organization:  []string{"Determined Master CA"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	_, signedCert, err := genKeyAndSignCert(caCert, caCert, key)
	if err != nil {
		return err
	}

	masterCACert = signedCert
	masterCAKey = key
	masterCAKeyBytes = x509.MarshalPKCS1PrivateKey(key)

	err = saveCACertAndKey()
	if err != nil {
		return err
	}

	return nil
}

// MasterCACert returns the CA cert.
func MasterCACert() ([]byte, error) {
	err := LoadOrGenCA()
	if err != nil {
		return nil, err
	}

	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: masterCACert.Raw,
	}
	return pem.EncodeToMemory(certBlock), nil
}

// MasterKeyAndCert returns the key and cert, signed by CA, that Master uses.
func MasterKeyAndCert() (keyPem []byte, certPem []byte, err error) {
	err = loadOrGenSignedMasterCert()
	if err != nil {
		return nil, nil, err
	}

	keyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(masterKey),
	}
	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: masterCert.Raw,
	}

	return pem.EncodeToMemory(keyBlock), pem.EncodeToMemory(certBlock), nil
}

// GenSignedCert generates a key and cert pair, signed by the master CA cert.
func GenSignedCert() (keyPem []byte, certPem []byte, err error) {
	if masterCAKey == nil || masterCACert == nil {
		return nil, nil, errors.New("unable to generate signed cert; generate master key and cert first")
	}

	random, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt))
	if err != nil {
		return nil, nil, err
	}
	cert := &x509.Certificate{
		SerialNumber: random,
		Subject: pkix.Name{
			Organization:  []string{"Determined Master"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
		},
		IPAddresses: []net.IP{},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(10, 0, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	certPrivKey, signedCert, err := genKeyAndSignCert(cert, masterCACert, masterCAKey)

	keyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	}
	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: signedCert.Raw,
	}

	return pem.EncodeToMemory(keyBlock), pem.EncodeToMemory(certBlock), err
}

// VerifyMasterSigned checks the offered certificate to ensure that it was signed by the master CA.
func VerifyMasterSigned(state tls.ConnectionState) error {
	if state.PeerCertificates != nil {
		for _, certificate := range state.PeerCertificates {
			err := certificate.CheckSignatureFrom(masterCACert)
			if err == nil {
				return nil
			}
		}
	}
	return fmt.Errorf("cert is not signed by master")
}
