package db

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
	"gopkg.in/oauth2.v3"

	"github.com/determined-ai/determined/master/pkg/model"
)

// ClientStore is a store for OAuth clients. It is separate from PgDB so we can implement an
// interface of the external OAuth library without polluting PgDB's method set.
type ClientStore struct {
	db *PgDB
}

// Create adds a new client to the database.
func (s *ClientStore) Create(c model.OAuthClient) error {
	return s.db.namedExecOne(`
INSERT INTO oauth.clients (id, secret, domain, name)
VALUES (:id, :secret, :domain, :name)`, c)
}

// List returns all OAuth clients in the database. The secrets are not included.
func (s *ClientStore) List() ([]model.OAuthClient, error) {
	rows, err := s.db.sql.Queryx(`
SELECT id, domain, name FROM oauth.clients
ORDER BY name`)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "querying OAuth clients")
	}
	defer rows.Close()

	clients := []model.OAuthClient{}
	for rows.Next() {
		var client model.OAuthClient
		if err = rows.StructScan(&client); err != nil {
			return nil, errors.Wrap(err, "scanning client")
		}
		clients = append(clients, client)
	}

	return clients, nil
}

func (s *ClientStore) remove(field, value string) error {
	_, err := s.db.sql.Exec(fmt.Sprintf(`DELETE FROM oauth.clients WHERE %s = $1`, field), value)
	if err == sql.ErrNoRows {
		return nil
	}
	return err
}

// RemoveByID removes the client with the given client ID.
func (s *ClientStore) RemoveByID(id string) error {
	return s.remove("id", id)
}

// GetByID returns a client given its ID, including the secret. It implements the
// gopkg.in/oauth2.v3#ClientStore interface, so it returns an external interface type; the returned
// object is always actually of type model.OAuthClient.
func (s *ClientStore) GetByID(id string) (oauth2.ClientInfo, error) {
	var c model.OAuthClient
	err := s.db.sql.Get(&c, `SELECT id, secret, domain, name FROM oauth.clients WHERE id = $1`, id)
	switch {
	case err == sql.ErrNoRows:
		return nil, errors.Errorf("unknown OAuth client ID %q", id)
	case err != nil:
		return nil, err
	}
	return c, nil
}

// ClientStore returns a store for OAuth clients backed by this database.
func (db *PgDB) ClientStore() *ClientStore {
	return &ClientStore{db}
}

// TokenStore is a store for OAuth tokens. It is separate from PgDB so we can implement an interface
// of the external OAuth library without polluting PgDB's method set.
type TokenStore struct {
	db *PgDB
}

// Create adds a new token to the database.
func (s *TokenStore) Create(info oauth2.TokenInfo) error {
	token := &model.OAuthToken{
		Access:           info.GetAccess(),
		AccessCreateAt:   info.GetAccessCreateAt(),
		AccessExpiresIn:  info.GetAccessExpiresIn(),
		ClientID:         info.GetClientID(),
		Code:             info.GetCode(),
		CodeCreateAt:     info.GetCodeCreateAt(),
		CodeExpiresIn:    info.GetCodeExpiresIn(),
		RedirectURI:      info.GetRedirectURI(),
		Refresh:          info.GetRefresh(),
		RefreshCreateAt:  info.GetRefreshCreateAt(),
		RefreshExpiresIn: info.GetRefreshExpiresIn(),
		Scope:            info.GetScope(),
		UserID:           info.GetUserID(),
	}

	return s.db.namedExecOne(`
INSERT INTO oauth.tokens
    (access, access_create_at, access_expires_in, client_id, code, code_create_at, code_expires_in,
    redirect_uri, refresh, refresh_create_at, refresh_expires_in, scope, user_id)
VALUES
    (:access, :access_create_at, :access_expires_in, :client_id, :code, :code_create_at,
    :code_expires_in, :redirect_uri, :refresh, :refresh_create_at, :refresh_expires_in, :scope,
    :user_id)`,
		token)
}

func (s *TokenStore) get(field, value string) (oauth2.TokenInfo, error) {
	if value == "" {
		return nil, nil
	}
	var token model.OAuthToken
	err := s.db.sql.Get(&token, fmt.Sprintf(`SELECT * FROM oauth.tokens WHERE %s = $1`, field), value)
	if err == sql.ErrNoRows {
		return nil, errors.Errorf("no token found with %s equal to %q", field, value)
	}
	return &token, err
}

func (s *TokenStore) remove(field, value string) error {
	_, err := s.db.sql.Exec(fmt.Sprintf(`DELETE FROM oauth.tokens WHERE %s = $1`, field), value)
	if err == sql.ErrNoRows {
		return nil
	}
	return err
}

// RemoveByCode deletes any tokens with the given authorization code.
func (s *TokenStore) RemoveByCode(code string) error { return s.remove("code", code) }

// RemoveByAccess deletes any tokens with the given access token value.
func (s *TokenStore) RemoveByAccess(access string) error { return s.remove("access", access) }

// RemoveByRefresh deletes any tokens with the given refresh token value.
func (s *TokenStore) RemoveByRefresh(refresh string) error { return s.remove("refresh", refresh) }

// GetByCode gets the token with the given authorization code.
func (s *TokenStore) GetByCode(code string) (oauth2.TokenInfo, error) { return s.get("code", code) }

// GetByAccess gets the token with the given access token value.
func (s *TokenStore) GetByAccess(access string) (oauth2.TokenInfo, error) {
	return s.get("access", access)
}

// GetByRefresh gets the token with the given refresh token value.
func (s *TokenStore) GetByRefresh(refresh string) (oauth2.TokenInfo, error) {
	return s.get("refresh", refresh)
}

// TokenStore returns a store for OAuth tokens backed by this database.
func (db *PgDB) TokenStore() *TokenStore {
	return &TokenStore{db}
}
