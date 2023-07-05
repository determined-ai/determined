package job

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// JobAuthZRBAC is basic OSS controls.
type JobAuthZRBAC struct{}

// FilterJobs returns a list of jobs that the user can view.
func (a *JobAuthZRBAC) FilterJobs(
	ctx context.Context, curUser model.User, jobs []*jobv1.Job,
) (viewableJobs []*jobv1.Job, err error) {
	viewableExpWorkspaces := make(map[int]bool)
	var viewableNtscWorkspaces map[model.AccessScopeID]bool
	hasNTSC := false
	hasExperiment := false
	userHasGlobalExpViewPerm := false
	userHasGlobalNTSCViewPerm := false
	for _, job := range jobs {
		switch job.Type {
		case jobv1.Type_TYPE_EXPERIMENT:
			hasExperiment = true
		case jobv1.Type_TYPE_NOTEBOOK, jobv1.Type_TYPE_TENSORBOARD, jobv1.Type_TYPE_SHELL,
			jobv1.Type_TYPE_COMMAND:
			hasNTSC = true
		}
		if hasNTSC && hasExperiment {
			break
		}
	}

	if hasExperiment {
		err = db.DoesPermissionMatch(ctx, curUser.ID, nil,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA)
		switch err.(type) {
		case nil:
			userHasGlobalExpViewPerm = true
		case authz.PermissionDeniedError:
			break
		default:
			return nil, err
		}
	}

	if hasNTSC {
		err = db.DoesPermissionMatch(ctx, curUser.ID, nil,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_NSC)
		switch err.(type) {
		case nil:
			userHasGlobalNTSCViewPerm = true
		case authz.PermissionDeniedError:
			break
		default:
			return nil, err
		}
	}

	if hasNTSC && !userHasGlobalNTSCViewPerm {
		viewableNtscWorkspaces, err = command.AuthZProvider.Get().
			AccessibleScopes(ctx, curUser, model.AccessScopeID(0))
		if err != nil {
			return nil, err
		}
	}

	if hasExperiment && !userHasGlobalExpViewPerm {
		viewableExpWorkspacesList, err := db.GetNonGlobalWorkspacesWithPermission(
			ctx, curUser.ID, rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA,
		)
		if err != nil {
			return nil, err
		}
		for _, workspaceID := range viewableExpWorkspacesList {
			viewableExpWorkspaces[workspaceID] = true
		}
	}

	viewableJobs = make([]*jobv1.Job, 0)
	for _, job := range jobs {
		switch job.Type {
		case jobv1.Type_TYPE_EXPERIMENT:
			viewable, _ := viewableExpWorkspaces[int(job.WorkspaceId)]
			if userHasGlobalExpViewPerm || viewable {
				viewableJobs = append(viewableJobs, job)
			}
		case jobv1.Type_TYPE_NOTEBOOK, jobv1.Type_TYPE_TENSORBOARD, jobv1.Type_TYPE_SHELL,
			jobv1.Type_TYPE_COMMAND:
			viewable, _ := viewableNtscWorkspaces[model.AccessScopeID(job.WorkspaceId)]
			if userHasGlobalNTSCViewPerm || viewable {
				viewableJobs = append(viewableJobs, job)
			}
			// TODO: special case for tensorboard.
		default:
			log.Warnf("ignoring job type: %s", job.Type)
			continue
		}
	}

	return viewableJobs, nil
}

// CanControlJobQueue returns an error if the user is not authorized to manipulate the
// job queue.
func (a *JobAuthZRBAC) CanControlJobQueue(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	return nil, nil
}

func init() {
	AuthZProvider.Register("rbac", &JobAuthZRBAC{})
}
