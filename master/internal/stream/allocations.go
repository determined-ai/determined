package stream

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
)

const (
	// AllocationsDeleteKey specifies the key for delete allocations.
	AllocationsDeleteKey = "allocations_deleted"
	// AllocationsUpsertKey specifies the key for upsert allocations.
	AllocationsUpsertKey = "allocation"
)

// AllocationMsg is a stream.Msg.
// determined:streamable
type AllocationMsg struct {
	bun.BaseModel `bun:"table:allocations"`

	ID     string      `bun:"allocation_id,pk" json:"id"`
	TaskID string      `bun:"task_id" json:"task_id"`
	Slots  int         `bun:"slots" json:"slots"`
	State  model.State `bun:"state" json:"state"`

	// XXX: time.Time freaks out because the allocations table stores timestamps without timezones
	StartTime string `bun:"start_time" json:"start_time"`
	EndTime   string `bun:"end_time" json:"end_time"`

	IsReady      bool   `bun:"is_ready" json:"is_ready"`
	Ports        JSONB  `bun:"ports" json:"ports"`
	ProxyAddress string `bun:"proxy_address" json:"proxy_address"`

	// metadata
	Seq int64 `bun:"seq" json:"seq"`

	// permission scope
	WorkspaceID int `json:"-"`

	// subscription scope
	ExperimentID int `bun:"experiment_id" json:"experiment_id"`
}

// SeqNum gets the SeqNum from a AllocationMsg.
func (am *AllocationMsg) SeqNum() int64 {
	return am.Seq
}

// UpsertMsg creates a Allocation stream upsert message.
func (am *AllocationMsg) UpsertMsg() stream.UpsertMsg {
	return stream.UpsertMsg{
		JSONKey: AllocationsUpsertKey,
		Msg:     am,
	}
}

// DeleteMsg creates a Allocation stream delete message.
func (am *AllocationMsg) DeleteMsg() stream.DeleteMsg {
	return stream.DeleteMsg{
		Key:     AllocationsDeleteKey,
		Deleted: am.ID,
	}
}

// AllocationSubscriptionSpec is what a user submits to define a allocation subscription.
// determined:streamable
type AllocationSubscriptionSpec struct {
	AllocationIds []string `json:"allocation_ids"`
	ExperimentIds []int    `json:"experiment_ids"`
	Since         int64    `json:"since"`
}

func getAllocationMsgsWithWorkspaceID(allocationMsgs []*AllocationMsg) *bun.SelectQuery {
	q := db.Bun().NewSelect().Model(&allocationMsgs).
		ColumnExpr("allocation_id").
		Column("task_id").
		Column("slots").
		Column("state").
		Column("start_time").
		Column("end_time").
		Column("is_ready").
		Column("ports").
		Column("proxy_address").
		Column("seq").
		Column("projects.workspace_id").
		Column("trials.experiment_id").
		Join("JOIN trial_id_task_id ON allocation_msg.task_id = trial_id_task_id.task_id").
		Join("JOIN trials ON trial_id_task_id.trial_id = trials.id").
		Join("JOIN experiments ON trials.experiment_id = experiments.id").
		Join("JOIN projects ON experiments.project_id = projects.id")
	return q
}

// AllocationCollectStartupMsgs collects AllocationMsg's that were missed prior to startup.
func AllocationCollectStartupMsgs(
	ctx context.Context,
	user model.User,
	known string,
	spec AllocationSubscriptionSpec,
) (
	[]stream.PreparableMessage, error,
) {
	var out []stream.PreparableMessage

	if len(spec.AllocationIds) == 0 && len(spec.ExperimentIds) == 0 {
		// empty subscription: everything known should be returned as deleted
		out = append(out, stream.DeleteMsg{
			Key:     AllocationsDeleteKey,
			Deleted: known,
		})
		return out, nil
	}
	// step 0: get user's permitted access scopes
	accessMap, err := AuthZProvider.Get().GetAllocationStreamableScopes(ctx, user)
	if err != nil {
		log.Debug("Issues with streamable scopes")
		return nil, err
	}
	var accessScopes []model.AccessScopeID
	for id, isPermitted := range accessMap {
		if isPermitted {
			accessScopes = append(accessScopes, id)
		}
	}

	permFilter := func(q *bun.SelectQuery) *bun.SelectQuery {
		if accessMap[model.GlobalAccessScopeID] {
			return q
		}
		return q.Where("workspace_id in (?)", bun.In(accessScopes))
	}

	// step 1: calculate all ids matching this subscription
	q := db.Bun().
		NewSelect().
		Table("allocations").
		ColumnExpr("allocations.allocation_id").
		Join("JOIN trial_id_task_id ON allocations.task_id = trial_id_task_id.task_id").
		Join("JOIN trials ON trial_id_task_id.trial_id = trials.id").
		Join("JOIN experiments ON trials.experiment_id = experiments.id").
		Join("JOIN projects ON experiments.project_id = projects.id").
		OrderExpr("allocation_id ASC")
	q = permFilter(q)

	// Ignore tmf.Since, because we want appearances, which might not be have seq > spec.Since.
	ws := stream.WhereSince{Since: 0}
	if len(spec.AllocationIds) > 0 {
		ws.Include("allocations.allocation_id in (?)", bun.In(spec.AllocationIds))
	}
	if len(spec.ExperimentIds) > 0 {
		ws.Include("experiment_id in (?)", bun.In(spec.ExperimentIds))
	}
	q = ws.Apply(q)

	var exist []string
	err = q.Scan(ctx, &exist)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		log.Errorf("error: %v\n", err)
		return nil, err
	}

	// step 2: figure out what was missing and what has appeared
	missing, appeared, err := stream.ProcessKnownString(known, exist)
	if err != nil {
		log.Debugf("failure trying to figure out what was missing and what appeared: %s", err)
		return nil, err
	}

	// step 3: hydrate appeared IDs into full AllocationMsgs
	var allocationMsgs []*AllocationMsg
	if len(appeared) > 0 {
		query := getAllocationMsgsWithWorkspaceID(allocationMsgs).
			Where("allocation_msg.allocation_id in (?)", bun.In(appeared))
		query = permFilter(query)
		err := query.Scan(ctx, &allocationMsgs)
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("error: %v\n", err)
			return nil, err
		}
	}

	// step 4: emit deletions and updates to the client
	out = append(out, stream.DeleteMsg{
		Key:     AllocationsDeleteKey,
		Deleted: missing,
	})
	for _, msg := range allocationMsgs {
		out = append(out, msg.UpsertMsg())
	}
	return out, nil
}

// AllocationCollectSubscriptionModMsgs scrapes the database when a
// user submits a new AllocationSubscriptionSpec for initial matches.
func AllocationCollectSubscriptionModMsgs(ctx context.Context, addSpec AllocationSubscriptionSpec) (
	[]interface{}, error,
) {
	if len(addSpec.AllocationIds) == 0 && len(addSpec.ExperimentIds) == 0 {
		return nil, nil
	}
	var allocationMsgs []*AllocationMsg
	q := getAllocationMsgsWithWorkspaceID(allocationMsgs)

	// Use WhereSince to build a complex WHERE clause.
	ws := stream.WhereSince{Since: addSpec.Since}
	if len(addSpec.AllocationIds) > 0 {
		ws.Include("id in (?)", bun.In(addSpec.AllocationIds))
	}
	if len(addSpec.ExperimentIds) > 0 {
		ws.Include("experiment_id in (?)", bun.In(addSpec.ExperimentIds))
	}
	q = ws.Apply(q)

	err := q.Scan(ctx)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		log.Errorf("error: %v\n", err)
		return nil, err
	}

	var out []interface{}
	for _, msg := range allocationMsgs {
		out = append(out, msg.UpsertMsg())
	}
	return out, nil
}

// AllocationFilterMaker tracks the allocation and experiment id's that are to be filtered for.
type AllocationFilterMaker struct {
	AllocationIds map[string]bool
	ExperimentIds map[int]bool
}

// NewAllocationFilterMaker creates a new FilterMaker.
func NewAllocationFilterMaker() FilterMaker[*AllocationMsg, AllocationSubscriptionSpec] {
	return &AllocationFilterMaker{make(map[string]bool), make(map[int]bool)}
}

// AddSpec adds AllocationIds and ExperimentIds specified in AllocationSubscriptionSpec.
func (as *AllocationFilterMaker) AddSpec(spec AllocationSubscriptionSpec) {
	for _, id := range spec.AllocationIds {
		as.AllocationIds[id] = true
	}
	for _, id := range spec.ExperimentIds {
		as.ExperimentIds[id] = true
	}
}

// DropSpec removes AllocationIds and ExperimentIds specified in AllocationSubscriptionSpec.
func (as *AllocationFilterMaker) DropSpec(spec AllocationSubscriptionSpec) {
	for _, id := range spec.AllocationIds {
		delete(as.AllocationIds, id)
	}
	for _, id := range spec.ExperimentIds {
		delete(as.ExperimentIds, id)
	}
}

// MakeFilter returns a function that determines if a AllocationMsg based on
// the AllocationFilterMaker's spec.
func (as *AllocationFilterMaker) MakeFilter() func(*AllocationMsg) bool {
	// Should this filter even run?
	if len(as.AllocationIds) == 0 && len(as.ExperimentIds) == 0 {
		return nil
	}

	// Make a copy of the maps, because the filter must run safely off-thread.
	allocationIds := make(map[string]bool)
	experimentIds := make(map[int]bool)
	for id := range as.AllocationIds {
		allocationIds[id] = true
	}
	for id := range as.ExperimentIds {
		experimentIds[id] = true
	}

	// return a closure around our copied maps
	return func(msg *AllocationMsg) bool {
		if _, ok := allocationIds[msg.ID]; ok {
			return true
		}
		if _, ok := experimentIds[msg.ExperimentID]; ok {
			return true
		}
		return false
	}
}

// AllocationMakePermissionFilter returns a function that checks if a AllocationMsg
// is in scope of the user permissions.
func AllocationMakePermissionFilter(ctx context.Context, user model.User) (func(*AllocationMsg) bool, error) {
	accessScopeSet, err := AuthZProvider.Get().GetAllocationStreamableScopes(ctx, user)
	if err != nil {
		return nil, err
	}

	switch {
	case accessScopeSet[model.GlobalAccessScopeID]:
		// user has global access for viewing allocations
		return func(msg *AllocationMsg) bool { return true }, nil
	default:
		return func(msg *AllocationMsg) bool {
			return accessScopeSet[model.AccessScopeID(msg.WorkspaceID)]
		}, nil
	}
}
