package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/token"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// PostAccessToken takes user id and optional lifespan, description and creates an
// access token for the given user.
func (a *apiServer) PostAccessToken(
	ctx context.Context, req *apiv1.PostAccessTokenRequest,
) (*apiv1.PostAccessTokenResponse, error) {
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

	tokenExpiration := token.DefaultTokenLifespan
	if req.Lifespan != "" {
		d, err := time.ParseDuration(req.Lifespan)
		if err != nil || d < 0 {
			return nil, status.Error(codes.InvalidArgument,
				"Lifespan must be a Go-formatted duration string with a positive value")
		}
		tokenExpiration = d
	}

	token, tokenID, err := token.CreateAccessToken(
		ctx, targetFullUser.ID, token.WithTokenExpiry(&tokenExpiration), token.WithTokenDescription(req.Description))
	if err != nil {
		return nil, err
	}
	return &apiv1.PostAccessTokenResponse{Token: token, TokenId: int32(tokenID)}, nil
}

type accessTokenFilter struct {
	OnlyActive bool            `json:"only_active"`
	Username   string          `json:"username"`
	TokenIDs   []model.TokenID `json:"token_ids"`
}

// GetAccessTokens returns all access token info.
func (a *apiServer) GetAccessTokens(
	ctx context.Context, req *apiv1.GetAccessTokensRequest,
) (*apiv1.GetAccessTokensResponse, error) {
	sortColMap := map[apiv1.GetAccessTokensRequest_SortBy]string{
		apiv1.GetAccessTokensRequest_SORT_BY_UNSPECIFIED: "id",
		apiv1.GetAccessTokensRequest_SORT_BY_USER_ID:     "user_id",
		apiv1.GetAccessTokensRequest_SORT_BY_EXPIRY:      "expiry",
		apiv1.GetAccessTokensRequest_SORT_BY_CREATED_AT:  "created_at",
		apiv1.GetAccessTokensRequest_SORT_BY_TOKEN_TYPE:  "token_type",
		apiv1.GetAccessTokensRequest_SORT_BY_REVOKED:     "revoked",
		apiv1.GetAccessTokensRequest_SORT_BY_DESCRIPTION: "description",
	}
	orderByMap := map[apiv1.OrderBy]string{
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
		Column("us.revoked").
		Column("us.description")

	var userIDForGivenUsername model.UserID
	if req.Filter != "" {
		var atf accessTokenFilter
		err := json.Unmarshal([]byte(req.Filter), &atf)
		if err != nil {
			return nil, err
		}

		if atf.Username != "" {
			err := db.Bun().NewSelect().
				Table("users").
				Column("id").
				Where("username = ?", atf.Username).
				Scan(ctx, &userIDForGivenUsername)
			if err != nil {
				return nil, err
			}
		}

		query = query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			if atf.OnlyActive {
				return q.Where("us.expiry > ?", time.Now().UTC()).
					Where("us.revoked = false")
			}
			return q
		}).WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			if userIDForGivenUsername > 0 {
				return q.Where("us.user_id = ?", userIDForGivenUsername)
			}
			return q
		}).WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			for _, tokenID := range atf.TokenIDs {
				if tokenID > 0 {
					q = q.WhereOr("us.id = ?", tokenID)
				}
			}
			return q
		})
	}

	// Get only Access token type
	query.Where("us.token_type = ?", model.TokenTypeAccessToken)

	orderBy, ok := orderByMap[req.OrderBy]
	if !ok {
		return nil, fmt.Errorf("unsupported order by %s", req.OrderBy)
	}
	sortColumn, ok := sortColMap[req.SortBy]
	if !ok {
		return nil, fmt.Errorf("unsupported sort by %s", req.SortBy)
	}
	query.OrderExpr("? ?", bun.Ident(sortColumn), bun.Safe(orderBy))
	if sortColumn != "id" {
		query.OrderExpr("id asc")
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	query, err = token.AuthZProvider.Get().CanGetAccessTokens(ctx, *curUser, query, userIDForGivenUsername)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
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
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	targetUserID, err := token.GetUserIDFromTokenID(ctx, req.TokenId)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Token with id: {%d} not found", req.TokenId))
	}
	err = token.AuthZProvider.Get().CanUpdateAccessToken(ctx, *curUser, targetUserID)
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
