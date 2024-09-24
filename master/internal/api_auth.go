package internal

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) Login(
	ctx context.Context, req *apiv1.LoginRequest,
) (*apiv1.LoginResponse, error) {
	if a.m.config.InternalConfig.ExternalSessions.JwtKey != "" {
		return nil, status.Error(codes.FailedPrecondition, "please run `det auth login` to authenticate")
	}

	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "missing argument: username")
	}

	userModel, err := user.ByUsername(ctx, req.Username)
	switch err {
	case nil:
	case db.ErrNotFound:
		return nil, grpcutil.ErrInvalidCredentials
	default:
		return nil, err
	}

	if userModel.Remote { // We can't return a more specific error for informational leak reasons.
		return nil, grpcutil.ErrInvalidCredentials
	}

	var hashedPassword string
	if req.IsHashed {
		hashedPassword = req.Password
	} else {
		hashedPassword = user.ReplicateClientSideSaltAndHash(req.Password)
	}

	if !userModel.ValidatePassword(hashedPassword) {
		return nil, grpcutil.ErrInvalidCredentials
	}

	if !userModel.Active {
		return nil, grpcutil.ErrNotActive
	}
	token, err := user.StartSession(ctx, userModel)
	if err != nil {
		return nil, err
	}
	fullUser, err := getUser(ctx, userModel.ID)
	return &apiv1.LoginResponse{Token: token, User: fullUser}, err
}

func (a *apiServer) CurrentUser(
	ctx context.Context, _ *apiv1.CurrentUserRequest,
) (*apiv1.CurrentUserResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	fullUser, err := getUser(ctx, user.ID)
	return &apiv1.CurrentUserResponse{User: fullUser}, err
}

func (a *apiServer) Logout(
	ctx context.Context, _ *apiv1.LogoutRequest,
) (*apiv1.LogoutResponse, error) {
	_, userSession, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if userSession == nil {
		return nil, status.Error(codes.InvalidArgument,
			"cannot manually logout of an allocation session")
	}

	// Do not want the AccessTokens to be deleted if the user logs out from session.
	// User can log back in using the same AccessToken: det u login <username> --token <token>
	if userSession.TokenType != model.TokenTypeAccessToken {
		err = user.DeleteSessionByID(ctx, userSession.ID)
	}
	return &apiv1.LogoutResponse{}, err
}

func redirectToLogin(c echo.Context) error {
	return c.Redirect(
		http.StatusSeeOther,
		fmt.Sprintf("/det/login?redirect=%s", c.Request().URL),
	)
}

// processProxyAuthentication is a middleware processing function that attempts
// to authenticate incoming HTTP requests coming through proxies.
func processProxyAuthentication(c echo.Context) (done bool, err error) {
	taskID := model.TaskID(strings.SplitN(c.Param("service"), ":", 2)[0])

	// Notebooks require special auth token passed as a URL parameter.
	token := extractNotebookTokenFromRequest(c.Request())
	var usr *model.User
	var notebookSession *model.NotebookSession

	if token != "" {
		// Notebooks go through special token param auth.
		usr, notebookSession, err = user.GetService().UserAndNotebookSessionFromToken(token)
		if err != nil {
			return true, err
		}
		if notebookSession.TaskID != taskID {
			return true, fmt.Errorf("invalid notebook session token for task (%v)", taskID)
		}
	} else {
		usr, _, err = user.GetService().UserAndSessionFromRequest(c.Request())
	}

	if errors.Is(err, db.ErrNotFound) {
		return true, redirectToLogin(c)
	} else if err != nil {
		return true, err
	}
	if !usr.Active {
		return true, redirectToLogin(c)
	}

	var ctx context.Context

	if c.Request() == nil || c.Request().Context() == nil {
		ctx = context.TODO()
	} else {
		ctx = c.Request().Context()
	}

	serviceNotFoundErr := api.NotFoundErrs("service", fmt.Sprint(taskID), false)

	spec, err := command.IdentifyTask(ctx, taskID)
	if errors.Is(err, db.ErrNotFound) || errors.Cause(err) == sql.ErrNoRows {
		// Check if it's an experiment.
		e, err := db.ExperimentByTaskID(ctx, taskID)
		if errors.Is(err, db.ErrNotFound) || errors.Cause(err) == sql.ErrNoRows {
			return true, err
		}

		if err != nil {
			return true, fmt.Errorf("error looking up task experiment: %w", err)
		}

		err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, *usr, e)
		return err != nil, authz.SubIfUnauthorized(err, serviceNotFoundErr)
	}

	if err != nil {
		return true, fmt.Errorf("error fetching task metadata: %w", err)
	}

	// Continue NTSC task checks.
	if spec.TaskType == model.TaskTypeTensorboard {
		err = command.AuthZProvider.Get().CanGetTensorboard(
			ctx, *usr, spec.WorkspaceID, spec.ExperimentIDs, spec.TrialIDs)
	} else {
		err = command.AuthZProvider.Get().CanGetNSC(
			ctx, *usr, spec.WorkspaceID)
	}
	return err != nil, authz.SubIfUnauthorized(err, serviceNotFoundErr)
}

// extractNotebookTokenFromRequest looks for auth token for Jupyter notebooks
// in two places:
// 1. A token query parameter in the request URL.
// 2. An HTTP Authorization header with a "token" type.
func extractNotebookTokenFromRequest(r *http.Request) string {
	token := r.URL.Query().Get("token")
	authRaw := r.Header.Get("Authorization")
	if token != "" {
		return token
	} else if authRaw != "" {
		if strings.HasPrefix(authRaw, "token ") {
			return strings.TrimPrefix(authRaw, "token ")
		}
	}
	// If we found no token, then abort the request with an HTTP 401.
	return ""
}

// processAuthWithRedirect is an auth middleware that redirects browser requests
// to login page for a set of given paths in case of authentication errors.
func processAuthWithRedirect(redirectPaths []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			echoErr := user.GetService().ProcessAuthentication(next)(c)
			if echoErr == nil {
				return nil
			} else if httpErr, ok := echoErr.(*echo.HTTPError); !ok || httpErr.Code != http.StatusUnauthorized {
				return echoErr
			}

			isProxiedPath := false
			path := c.Request().RequestURI
			for _, p := range redirectPaths {
				if strings.HasPrefix(path, p) {
					isProxiedPath = true
				}
			}
			if !isProxiedPath {
				// GRPC-backed routes are authenticated by grpcutil.*AuthInterceptor.
				return echoErr
			}

			md := metadata.MD{}
			for k, v := range c.Request().Header {
				k = strings.TrimPrefix(k, grpcutil.GrpcMetadataPrefix)
				md.Append(k, v...)
			}
			_, _, err := grpcutil.GetUser(metadata.NewIncomingContext(c.Request().Context(), md))
			if err == nil {
				return next(c)
			}
			errStatus := status.Convert(err)
			if errStatus.Code() != codes.PermissionDenied && errStatus.Code() != codes.Unauthenticated {
				return err
			}

			// TODO: reverse this logic to redirect only if accept is empty or specifies text/html.
			// No web page redirects for programmatic requests.
			for _, accept := range c.Request().Header["Accept"] {
				if strings.Contains(accept, "application/json") {
					return echoErr
				}
			}

			return redirectToLogin(c)
		}
	}
}
