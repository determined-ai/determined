package model

import (
	"time"

	"gopkg.in/oauth2.v3"
)

// OAuthClient represents one OAuth client application.
type OAuthClient struct {
	ID     string `db:"id" json:"id"`
	Secret string `db:"secret" json:"secret"`
	Domain string `db:"domain" json:"domain"`
	Name   string `db:"name" json:"name"`
}

// GetID implements the oauth2.ClientInfo interface.
func (c OAuthClient) GetID() string { return c.ID }

// GetSecret implements the oauth2.ClientInfo interface.
func (c OAuthClient) GetSecret() string { return c.Secret }

// GetDomain implements the oauth2.ClientInfo interface.
func (c OAuthClient) GetDomain() string { return c.Domain }

// GetUserID implements the oauth2.ClientInfo interface.
func (c OAuthClient) GetUserID() string { return "" }

// OAuthToken represents an OAuth token.
type OAuthToken struct {
	Access           string        `db:"access" json:"access"`
	AccessCreateAt   time.Time     `db:"access_create_at" json:"access_create_at"`
	AccessExpiresIn  time.Duration `db:"access_expires_in" json:"access_expires_in"`
	ClientID         string        `db:"client_id" json:"client_id"`
	Code             string        `db:"code" json:"code"`
	CodeCreateAt     time.Time     `db:"code_create_at" json:"code_create_at"`
	CodeExpiresIn    time.Duration `db:"code_expires_in" json:"code_expires_in"`
	RedirectURI      string        `db:"redirect_uri" json:"redirect_uri"`
	Refresh          string        `db:"refresh" json:"refresh"`
	RefreshCreateAt  time.Time     `db:"refresh_create_at" json:"refresh_create_at"`
	RefreshExpiresIn time.Duration `db:"refresh_expires_in" json:"refresh_expires_in"`
	Scope            string        `db:"scope" json:"scope"`
	UserID           string        `db:"user_id" json:"user_id"`

	ID int `db:"id" json:"id"`
}

// New create to token model instance.
func (t *OAuthToken) New() oauth2.TokenInfo { return &OAuthToken{} }

// GetClientID the client id.
func (t *OAuthToken) GetClientID() string { return t.ClientID }

// SetClientID the client id.
func (t *OAuthToken) SetClientID(clientID string) { t.ClientID = clientID }

// GetUserID the user id.
func (t *OAuthToken) GetUserID() string { return t.UserID }

// SetUserID the user id.
func (t *OAuthToken) SetUserID(userID string) { t.UserID = userID }

// GetRedirectURI redirect URI.
func (t *OAuthToken) GetRedirectURI() string { return t.RedirectURI }

// SetRedirectURI redirect URI.
func (t *OAuthToken) SetRedirectURI(redirectURI string) { t.RedirectURI = redirectURI }

// GetScope get scope of authorization.
func (t *OAuthToken) GetScope() string { return t.Scope }

// SetScope get scope of authorization.
func (t *OAuthToken) SetScope(scope string) { t.Scope = scope }

// GetCode authorization code.
func (t *OAuthToken) GetCode() string { return t.Code }

// SetCode authorization code.
func (t *OAuthToken) SetCode(code string) { t.Code = code }

// GetCodeCreateAt create Time.
func (t *OAuthToken) GetCodeCreateAt() time.Time { return t.CodeCreateAt }

// SetCodeCreateAt create Time.
func (t *OAuthToken) SetCodeCreateAt(createAt time.Time) { t.CodeCreateAt = createAt }

// GetCodeExpiresIn the lifetime in seconds of the authorization code.
func (t *OAuthToken) GetCodeExpiresIn() time.Duration { return t.CodeExpiresIn }

// SetCodeExpiresIn the lifetime in seconds of the authorization code.
func (t *OAuthToken) SetCodeExpiresIn(exp time.Duration) { t.CodeExpiresIn = exp }

// GetAccess access Token.
func (t *OAuthToken) GetAccess() string { return t.Access }

// SetAccess access Token.
func (t *OAuthToken) SetAccess(access string) { t.Access = access }

// GetAccessCreateAt create Time.
func (t *OAuthToken) GetAccessCreateAt() time.Time { return t.AccessCreateAt }

// SetAccessCreateAt create Time.
func (t *OAuthToken) SetAccessCreateAt(createAt time.Time) { t.AccessCreateAt = createAt }

// GetAccessExpiresIn the lifetime in seconds of the access token.
func (t *OAuthToken) GetAccessExpiresIn() time.Duration { return t.AccessExpiresIn }

// SetAccessExpiresIn the lifetime in seconds of the access token.
func (t *OAuthToken) SetAccessExpiresIn(exp time.Duration) { t.AccessExpiresIn = exp }

// GetRefresh refresh Token.
func (t *OAuthToken) GetRefresh() string { return t.Refresh }

// SetRefresh refresh Token.
func (t *OAuthToken) SetRefresh(refresh string) { t.Refresh = refresh }

// GetRefreshCreateAt create Time.
func (t *OAuthToken) GetRefreshCreateAt() time.Time { return t.RefreshCreateAt }

// SetRefreshCreateAt create Time.
func (t *OAuthToken) SetRefreshCreateAt(createAt time.Time) { t.RefreshCreateAt = createAt }

// GetRefreshExpiresIn the lifetime in seconds of the refresh token.
func (t *OAuthToken) GetRefreshExpiresIn() time.Duration { return t.RefreshExpiresIn }

// SetRefreshExpiresIn the lifetime in seconds of the refresh token.
func (t *OAuthToken) SetRefreshExpiresIn(exp time.Duration) { t.RefreshExpiresIn = exp }
