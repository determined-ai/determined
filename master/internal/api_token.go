package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/license"
	"github.com/determined-ai/determined/master/internal/token"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

var errAccessTokenRequiresEE = status.Error(
	codes.FailedPrecondition,
	"users cannot log in with an access token without a valid Enterprise Edition license set up.",
)

// PostAccessToken takes user id and optional lifespan, description and creates an
// access token for the given user.
func (a *apiServer) PostAccessToken(
	ctx context.Context, req *apiv1.PostAccessTokenRequest,
) (*apiv1.PostAccessTokenResponse, error) {
	if !license.IsEE() {
		return nil, errAccessTokenRequiresEE
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	targetFullUser, err := getFullModelUser(ctx, model.UserID(req.UserId))
	if err != nil {
		return nil, err
	}
	targetUser := targetFullUser.ToUser()
	if err = token.AuthZProvider.Get().CanCreateAccessToken(ctx, *curUser, targetUser); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	maxTokenLifespan := a.m.config.Security.Token.MaxLifespan()
	tokenExpiration := a.m.config.Security.Token.DefaultLifespan()
	if req.Lifespan != nil {
		if *req.Lifespan == config.InfiniteTokenLifespanString {
			tokenExpiration = maxTokenLifespan
		} else {
			d, err := time.ParseDuration(*req.Lifespan)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument,
					"failed to parse lifespan %s: %s", *req.Lifespan, err)
			} else if d < 0 {
				return nil, status.Error(codes.InvalidArgument,
					"lifespan must be a Go-formatted duration string with a positive value")
			}
			tokenExpiration = d
		}
	}

	// Ensure the token lifespan does not exceed the maximum allowed or minimum -1 value.
	if tokenExpiration > maxTokenLifespan {
		return nil, status.Error(codes.InvalidArgument, "token Lifespan must be less than max token lifespan")
	}
	if tokenExpiration < 0 {
		return nil, status.Error(codes.InvalidArgument, "token lifespan must be greater than 0 days,"+
			" unless set to -1 for infinite lifespan")
	}

	token, tokenID, err := token.CreateAccessToken(
		ctx, targetFullUser.ID, token.WithTokenExpiry(&tokenExpiration), token.WithTokenDescription(req.Description))
	if err != nil {
		return nil, err
	}
	return &apiv1.PostAccessTokenResponse{Token: token, TokenId: int32(tokenID)}, nil
}

// GetAccessTokens returns all access token info.
func (a *apiServer) GetAccessTokens(
	ctx context.Context, req *apiv1.GetAccessTokensRequest,
) (*apiv1.GetAccessTokensResponse, error) {
	if !license.IsEE() {
		return nil, errAccessTokenRequiresEE
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	sortCols := map[apiv1.GetAccessTokensRequest_SortBy]string{
		apiv1.GetAccessTokensRequest_SORT_BY_UNSPECIFIED: "id",
		apiv1.GetAccessTokensRequest_SORT_BY_USER_ID:     "user_id",
		apiv1.GetAccessTokensRequest_SORT_BY_EXPIRY:      "expiry",
		apiv1.GetAccessTokensRequest_SORT_BY_CREATED_AT:  "created_at",
		apiv1.GetAccessTokensRequest_SORT_BY_TOKEN_TYPE:  "token_type",
		apiv1.GetAccessTokensRequest_SORT_BY_REVOKED:     "revoked_at",
		apiv1.GetAccessTokensRequest_SORT_BY_DESCRIPTION: "description",
	}
	orderDirections := map[apiv1.OrderBy]string{
		apiv1.OrderBy_ORDER_BY_UNSPECIFIED: "ASC",
		apiv1.OrderBy_ORDER_BY_ASC:         "ASC",
		apiv1.OrderBy_ORDER_BY_DESC:        "DESC",
	}

	userSessions := []model.UserSession{}

	query := db.Bun().NewSelect().Model(&userSessions).
		ModelTableExpr("user_sessions as us").
		Column("us.id").
		Column("us.user_id").
		Column("us.expiry").
		Column("us.created_at").
		Column("us.token_type").
		Column("us.revoked_at").
		Column("us.description")

	var userIDForGivenUsername model.UserID

	if req.Username != "" {
		err := db.Bun().NewSelect().
			Table("users").
			Column("id").
			Where("username = ?", req.Username).
			Scan(ctx, &userIDForGivenUsername)
		if err != nil {
			return nil, err
		}
		if userIDForGivenUsername > 0 {
			query.Where("us.user_id = ?", userIDForGivenUsername)
		}
	}

	// CanGetAccessTokens ensures that the logged-in user has the required permissions
	// to perform actions on the target user's account.
	query, err = token.AuthZProvider.Get().CanGetAccessTokens(ctx, *curUser, query, &userIDForGivenUsername)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if !req.ShowInactive {
		query.Where("us.expiry > ?", time.Now().UTC()).
			Where("us.revoked_at IS NULL")
	}

	// Get only Access token type
	query.Where("us.token_type = ?", model.TokenTypeAccessToken)

	if len(req.TokenIds) > 0 {
		query = query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			for _, tokenID := range req.TokenIds {
				if tokenID > 0 {
					q = q.WhereOr("us.id = ?", tokenID)
				}
			}
			return q
		})
	}

	orderBy, ok := orderDirections[req.OrderBy]
	if !ok {
		return nil, fmt.Errorf("unsupported order by %s", req.OrderBy)
	}
	sortColumn, ok := sortCols[req.SortBy]
	if !ok {
		return nil, fmt.Errorf("unsupported sort by %s", req.SortBy)
	}
	query.OrderExpr("? ?", bun.Ident(sortColumn), bun.Safe(orderBy))
	if sortColumn != "id" {
		query.OrderExpr("id asc")
	}

	err = query.Scan(ctx)
	if err != nil {
		return nil, err
	}

	res := &apiv1.GetAccessTokensResponse{}
	for _, s := range userSessions {
		res.TokenInfo = append(res.TokenInfo, s.Proto())
	}
	return res, api.Paginate(&res.Pagination, &res.TokenInfo, req.Offset, req.Limit)
}

// PatchAccessToken performs a partial patch of mutable fields on an existing access token.
func (a *apiServer) PatchAccessToken(
	ctx context.Context, req *apiv1.PatchAccessTokenRequest,
) (*apiv1.PatchAccessTokenResponse, error) {
	if !license.IsEE() {
		return nil, errAccessTokenRequiresEE
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	targetUserID, err := token.GetUserIDFromTokenID(ctx, req.TokenId)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Token with id: {%d} not found", req.TokenId))
	}
	err = token.AuthZProvider.Get().CanUpdateAccessToken(ctx, *curUser, *targetUserID)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	patchOptions := token.AccessTokenUpdateOptions{
		Description: req.Description,
		SetRevoked:  req.SetRevoked,
	}
	patchedTokenInfo, err := token.UpdateAccessToken(ctx, model.TokenID(req.TokenId), patchOptions)
	if err != nil {
		return nil, err
	}
	return &apiv1.PatchAccessTokenResponse{TokenInfo: patchedTokenInfo.Proto()}, nil
}
