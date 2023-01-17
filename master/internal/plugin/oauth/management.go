package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/scrypt"

	"github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/pkg/model"
)

// addClient is the handler for adding new OAuth clients. It is the only way to get the secret for
// an application.
func (s *Service) addClient(c echo.Context) error {
	// To lock things down a bit and reduce the need for authorization UI, only allow a single client.
	clients, err := s.clientStore.List()
	if err != nil {
		return err
	}
	if len(clients) > 0 {
		return errors.New("only one OAuth client is allowed")
	}

	request := struct {
		Domain string `json:"domain"`
		Name   string `json:"name"`
	}{}
	dec := json.NewDecoder(c.Request().Body)
	if err = dec.Decode(&request); err != nil {
		return errors.Wrap(err, "failed to parse request body")
	}
	if request.Domain == "" {
		return errors.New("missing domain")
	}
	if request.Name == "" {
		return errors.New("missing name")
	}

	user := c.(*context.DetContext).MustGetUser()

	log.WithFields(log.Fields{
		"domain": request.Domain,
		"name":   request.Name,
		"user":   user.Username,
	}).Info("adding new OAuth client")

	idBytes := make([]byte, 32)
	if _, err = rand.Read(idBytes); err != nil {
		return errors.Wrap(err, "failed to generate client ID")
	}

	secretBytes := make([]byte, 32)
	if _, err = rand.Read(secretBytes); err != nil {
		return errors.Wrap(err, "failed to generate client secret")
	}

	client := model.OAuthClient{
		ID:     hex.EncodeToString(idBytes),
		Domain: request.Domain,
		Name:   request.Name,
	}

	secret := hex.EncodeToString(secretBytes)

	// Store a hashed secret in the database and return the raw secret.
	if client.Secret, err = hashSecret(client.ID, secret); err != nil {
		return errors.Wrap(err, "failed to hash secret")
	}

	if err := s.clientStore.Create(client); err != nil {
		return errors.Wrap(err, "failed to store new client")
	}
	client.Secret = secret
	return c.JSON(http.StatusOK, client)
}

func (s *Service) clients(c echo.Context) error {
	clients, err := s.clientStore.List()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, clients)
}

func (s *Service) deleteClient(c echo.Context) error {
	clientID := c.Param("id")
	log.WithField("client_id", clientID).Info("deleting OAuth client")
	return s.clientStore.RemoveByID(c.Param("id"))
}

func hashSecret(id, secret string) (string, error) {
	// The difficulty settings are a step up from the recommendation as of 2017 (see
	// https://pkg.go.dev/golang.org/x/crypto/scrypt).
	key, err := scrypt.Key([]byte(id), []byte(secret), 1<<16, 8, 1, 32)
	return hex.EncodeToString(key), err
}

// ValidateRequest checks whether the given request contains valid OAuth credentials.
func (s *Service) ValidateRequest(c echo.Context) (bool, error) {
	authHeader := c.Request().Header.Get("Authorization")
	bearer := strings.TrimPrefix(authHeader, "Bearer ")
	if bearer == authHeader {
		return false, nil
	}

	token, err := s.server.Manager.LoadAccessToken(bearer)
	if err != nil {
		logrus.WithError(err).Error("failed to load access token")
		return false, nil
	}
	return token != nil, nil
}
