package user

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

func getOrCreateKeys(db *db.PgDB) (*model.AuthTokenKeypair, error) {
	switch storedKeys, err := db.AuthTokenKeypair(); {
	case err != nil:
		return nil, errors.Wrap(err, "error retrieving auth token keypair")
	case storedKeys == nil:
		publicKey, privateKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			return nil, errors.Wrap(err, "error creating auth token keypair")
		}
		tokenKeypair := model.AuthTokenKeypair{PublicKey: publicKey, PrivateKey: privateKey}
		err = db.AddAuthTokenKeypair(&tokenKeypair)
		if err != nil {
			return nil, errors.Wrap(err, "error saving auth token keypair")
		}
		return &tokenKeypair, nil
	default:
		return storedKeys, nil
	}
}
