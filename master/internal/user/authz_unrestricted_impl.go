package user

/*
import (
	"reflect"
	"runtime"
	"strings"

	"github.com/determined-ai/determined/master/pkg/model"
)

// We litterally made our own mock lol.
// Convert this to mocketio.
// An error that
type blockingFunc struct {
	Func any
}

func functionName(fn any) string {
	longName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	// Converts the name from this long version to the more readable version of.
	// github.com/determined-ai/determined/master/internal/user.UserAuthZ.CanGetMe-fm
	// .CanGetMe-fm
	return longName[strings.LastIndex(longName, "."):]
}

func (f blockingFunc) Error() string {
	return functionName(f.Func)
}

// UserAuthZUnrestricted is unrestricted.
// TODO XXX: this is for demo purposes only. Remove before merging to master branch.
type UserAuthZUnrestricted struct {
	AlwaysAllow bool
}

// CanGetUser for unresticted authz.
func (a *UserAuthZUnrestricted) CanGetUser(currentUser, targetUser model.User) error {
	return blockingFunc{a.CanGetUser}
}

// FilterUseList for unresticted authz.
func (a *UserAuthZUnrestricted) FilterUserList(
	curUser model.User, users []model.FullUser,
) ([]model.FullUser, error) {
	return nil, blockingFunc{a.FilterUserList}
}

// CanCreateUser for unresticted authz.
func (a *UserAuthZUnrestricted) CanCreateUser(
	curUser, userToAdd model.User, agentUserGroup *model.AgentUserGroup,
) error {
	return blockingFunc{a.CanCreateUser}
}

// CanGetMe for unresticted authz.
func (a *UserAuthZUnrestricted) CanGetMe(curUser model.User) error {
	return blockingFunc{a.CanGetMe}
}

// CanSetUserPassword for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUsersPassword(currentUser, targetUser model.User) error {
	return blockingFunc{a.CanSetUsersPassword}
}

// CanSetUsersActive for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUsersActive(
	curUser, userToAdd model.User, toActiveVal bool,
) error {
	return blockingFunc{a.CanSetUsersActive}
}

// CanSetUsersAdmin for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUsersAdmin(
	curUser, userToAdd model.User, toAdminVal bool,
) error {
	return blockingFunc{a.CanSetUsersAdmin}
}

// CanSetUsersAgentUserGroup for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUsersAgentUserGroup(
	curUser, userToAdd model.User, agentUserGroup model.AgentUserGroup,
) error {
	return blockingFunc{a.CanSetUsersAgentUserGroup}
}

// CanSetUsersUsername for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUsersUsername(curUser, targetUser model.User) error {
	return blockingFunc{a.CanSetUsersUsername}
}

// CanSetUsersDisplayName for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUsersDisplayName(curUser, targetUser model.User) error {
	return blockingFunc{a.CanSetUsersDisplayName}
}

// CanGetUsersImage for unresticted authz.
func (a *UserAuthZUnrestricted) CanGetUsersImage(curUser model.User, targetUsername string) error {
	return blockingFunc{a.CanGetUsersImage}
}

// CanGetUsersOwnSettings for unresticted authz.
func (a *UserAuthZUnrestricted) CanGetUsersOwnSettings(curUser model.User) error {
	return blockingFunc{a.CanGetUsersOwnSettings}
}

// CanCreateUsersOwnSetting for unresticted authz.
func (a *UserAuthZUnrestricted) CanCreateUsersOwnSetting(
	curUser model.User, setting model.UserWebSetting,
) error {
	return blockingFunc{a.CanCreateUsersOwnSetting}
}

// CanResetUsersOwnSettings for unresticted authz.
func (a *UserAuthZUnrestricted) CanResetUsersOwnSettings(curUser model.User) error {
	return blockingFunc{a.CanResetUsersOwnSettings}
}

func init() {
	AuthZProvider.Register("unrestricted", &UserAuthZUnrestricted{AlwaysAllow: true})
	AuthZProvider.Register("restricted", &UserAuthZUnrestricted{AlwaysAllow: false})
}
*/
