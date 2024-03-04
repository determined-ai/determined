package multirm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

const rp = "resource-pool"

func TestGetAllocationSummaries(t *testing.T) {
	cases := []struct {
		name       string
		allocNames []string
		managers   int
	}{
		{"simple", []string{uuid.NewString(), uuid.NewString()}, 1},
		{"multirm", []string{uuid.NewString(), uuid.NewString()}, 3},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rms := map[string]rm.ResourceManager{}
			for i := 1; i <= tt.managers; i++ {
				ret := map[model.AllocationID]sproto.AllocationSummary{}
				for _, alloc := range tt.allocNames {
					a := alloc + fmt.Sprint(i)
					ret[*model.NewAllocationID(&a)] = sproto.AllocationSummary{}
				}
				require.Equal(t, len(ret), len(tt.allocNames))

				mockRM := mocks.ResourceManager{}
				mockRM.On("GetAllocationSummaries").Return(ret, nil)

				rms[uuid.NewString()] = &mockRM
			}

			m := &MultiRMRouter{rms: rms}

			allocs, err := m.GetAllocationSummaries()
			require.NoError(t, err)
			require.Equal(t, tt.managers*len(tt.allocNames), len(allocs))
			require.NotNil(t, allocs)

			bogus := "bogus"
			require.Empty(t, allocs[*model.NewAllocationID(&bogus)])

			for _, name := range tt.allocNames {
				n := fmt.Sprintf(name + "0")
				tmpName := name

				require.NotNil(t, allocs[*model.NewAllocationID(&n)])
				require.Empty(t, allocs[*model.NewAllocationID(&tmpName)])
			}
		})
	}
}

func TestAllocate(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")

	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	allocReq := sproto.AllocateRequest{ResourceManager: manager}
	mockRM.On("Allocate", manager, allocReq).Return(&sproto.ResourcesSubscription{}, nil)

	res, err := m.Allocate(manager, allocReq)
	require.NoError(t, err)
	require.Equal(t, res, &sproto.ResourcesSubscription{})

	// Check that bogus RM call errors.
	res, err = m.Allocate("bogus", allocReq)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
	require.Nil(t, res)
}

func TestValidateResources(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	req := sproto.ValidateResourcesRequest{
		ResourcePool: "",
		Slots:        0,
		IsSingleNode: true,
	}

	mockRM.On("ValidateResources", manager, req).Return(nil, nil)

	_, err := m.ValidateResources(manager, req)
	require.NoError(t, err)

	// Check that bogus RM call errors.
	_, err = m.ValidateResources("bogus", req)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
}

func TestDeleteJob(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}
	job1 := sproto.DeleteJob{JobID: model.JobID("job1")}

	mockRM.On("DeleteJob", job1).Return(sproto.EmptyDeleteJobResponse(), nil)

	_, err := m.DeleteJob(job1)
	require.NoError(t, err)
}

func TestNotifyContainerRunning(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	mockRM.On("NotifyContainerRunning", manager, sproto.NotifyContainerRunning{}).Return(nil)

	err := m.NotifyContainerRunning(sproto.NotifyContainerRunning{})
	require.Equal(t, err, rmerrors.ErrNotSupported)
}

func TestSetGroupWeight(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	req1 := sproto.SetGroupWeight{ResourcePool: "rp1"}

	mockRM.On("SetGroupWeight", manager, req1).Return(nil)

	err := m.SetGroupWeight(manager, req1)
	require.NoError(t, err)

	// Check that bogus RM call errors.
	err = m.SetGroupWeight("bogus", req1)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
}

func TestSetGroupPriority(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	req1 := sproto.SetGroupPriority{ResourcePool: "rp1"}

	mockRM.On("SetGroupPriority", manager, req1).Return(nil)

	err := m.SetGroupPriority(manager, req1)
	require.NoError(t, err)

	// Check that bogus RM call errors.
	err = m.SetGroupPriority("bogus", req1)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
}

func TestExternalPreemptionPending(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	alloc1 := model.AllocationID("alloc1")

	mockRM.On("ExternalPreemptionPending", manager, alloc1).Return(nil)

	err := m.ExternalPreemptionPending(alloc1)
	require.Equal(t, err, rmerrors.ErrNotSupported)
}

func TestIsReattachable(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	mockRM.On("IsReattachableOnlyAfterStarted", mock.Anything).Return(true)

	val := m.IsReattachableOnlyAfterStarted(manager)
	require.Equal(t, true, val)

	// Check that bogus RM call defaults to false.
	val = m.IsReattachableOnlyAfterStarted("bogus")
	require.Equal(t, false, val)
}

func TestGetResourcePools(t *testing.T) {
	cases := []struct {
		name     string
		rpNames  []string
		managers int
	}{
		{"simple", []string{uuid.NewString(), uuid.NewString()}, 1},
		{"multirm", []string{uuid.NewString(), uuid.NewString()}, 5},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rms := map[string]rm.ResourceManager{}
			for i := 1; i <= tt.managers; i++ {
				ret := []*resourcepoolv1.ResourcePool{}
				for _, n := range tt.rpNames {
					ret = append(ret, &resourcepoolv1.ResourcePool{Name: n})
				}

				mockRM := mocks.ResourceManager{}
				mockRM.On("GetResourcePools").Return(&apiv1.GetResourcePoolsResponse{ResourcePools: ret}, nil)

				rms[uuid.NewString()] = &mockRM
			}

			m := &MultiRMRouter{rms: rms}

			rps, err := m.GetResourcePools()
			require.NoError(t, err)
			require.Equal(t, tt.managers*len(tt.rpNames), len(rps.ResourcePools))

			for _, rp := range rps.ResourcePools {
				require.Contains(t, tt.rpNames, rp.Name)
			}
		})
	}
}

func TestGetDefaultResourcePools(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	res1 := sproto.GetDefaultComputeResourcePoolResponse{PoolName: "default"}
	res2 := sproto.GetDefaultAuxResourcePoolResponse{PoolName: "default"}

	mockRM.On("GetDefaultComputeResourcePool", manager).Return(res1, nil)
	mockRM.On("GetDefaultAuxResourcePool", manager).Return(res2, nil)

	actual1, err := m.GetDefaultComputeResourcePool(manager)
	require.NoError(t, err)
	require.Equal(t, res1, actual1)

	actual2, err := m.GetDefaultAuxResourcePool(manager)
	require.NoError(t, err)
	require.Equal(t, res2, actual2)

	// Check that bogus RM call errors.
	actual1, err = m.GetDefaultComputeResourcePool("bogus")
	require.Equal(t, err, ErrRMNotDefined("bogus"))
	require.Empty(t, actual1)

	// Check that bogus RM call errors.
	actual2, err = m.GetDefaultAuxResourcePool("bogus")
	require.Equal(t, err, ErrRMNotDefined("bogus"))
	require.Empty(t, actual2)
}

func TestValidateResourcePool(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	mockRM.On("ValidateResourcePool", manager, rp).Return(nil)

	err := m.ValidateResourcePool(manager, rp)
	require.NoError(t, err)

	// Check that bogus RM call errors.
	err = m.ValidateResourcePool("bogus", rp)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
}

func TestResolveResourcePool(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	req := sproto.ResolveResourcesRequest{ResourcePool: rp}

	mockRM.On("ResolveResourcePool", manager, req).Return(manager, rp, nil)

	resolvedRM, resolvedRP, err := m.ResolveResourcePool(manager, req)
	require.NoError(t, err)
	require.Equal(t, manager, resolvedRM)
	require.Equal(t, rp, resolvedRP)

	// Check that bogus RM call errors.
	resolvedRM, resolvedRP, err = m.ResolveResourcePool("bogus", req)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
	require.Equal(t, "bogus", resolvedRM)
	require.Equal(t, rp, resolvedRP)
}

func TestTaskContainerDefaults(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	res := model.TaskContainerDefaultsConfig{}

	mockRM.On("TaskContainerDefaults", manager, rp, res).Return(res, nil)

	_, err := m.TaskContainerDefaults(manager, rp, res)
	require.NoError(t, err)

	// Check that bogus RM call errors.
	_, err = m.TaskContainerDefaults("bogus", rp, res)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
}

func TestGetJobQ(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	res := map[model.JobID]*sproto.RMJobInfo{}

	mockRM.On("GetJobQ", manager, rp).Return(res, nil)

	ret, err := m.GetJobQ(manager, rp)
	require.NoError(t, err)
	require.Equal(t, ret, res)

	// Check that bogus RM call errors.
	ret, err = m.GetJobQ("bogus", rp)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
	require.Empty(t, ret)
}

func TestGetJobQueueStatsRequest(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	req := &apiv1.GetJobQueueStatsRequest{ResourcePools: []string{rp}}
	res := &apiv1.GetJobQueueStatsResponse{}

	mockRM.On("GetJobQueueStatsRequest", manager, req).Return(res, nil)

	ret, err := m.GetJobQueueStatsRequest(manager, req)
	require.NoError(t, err)
	require.Equal(t, ret, res)

	// Check that bogus RM call errors.
	ret, err = m.GetJobQueueStatsRequest("bogus", req)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
	require.Empty(t, ret)
}

func TestMoveJob(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	req := sproto.MoveJob{ResourcePool: rp}

	mockRM.On("MoveJob", manager, req).Return(nil)

	err := m.MoveJob(manager, req)
	require.NoError(t, err)

	// Check that bogus RM call errors.
	err = m.MoveJob("bogus", req)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
}

func TestGetExternalJobs(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	res := []*jobv1.Job{}

	mockRM.On("GetExternalJobs", manager, rp).Return(res, nil)

	ret, err := m.GetExternalJobs(manager, rp)
	require.NoError(t, err)
	require.Equal(t, ret, res)

	// Check that bogus RM call errors.
	ret, err = m.GetExternalJobs("bogus", rp)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
	require.Empty(t, ret)
}

func TestGetAgents(t *testing.T) {
	cases := []struct {
		name       string
		agentNames []string
		managers   int
	}{
		{"simple", []string{uuid.NewString(), uuid.NewString()}, 1},
		{"multirm", []string{uuid.NewString(), uuid.NewString()}, 5},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rms := map[string]rm.ResourceManager{}
			for i := 1; i <= tt.managers; i++ {
				ret := []*agentv1.Agent{}
				for _, n := range tt.agentNames {
					ret = append(ret, &agentv1.Agent{ResourcePools: []string{n}})
				}

				mockRM := mocks.ResourceManager{}
				mockRM.On("GetAgents").Return(&apiv1.GetAgentsResponse{Agents: ret}, nil)

				rms[uuid.NewString()] = &mockRM
			}

			m := &MultiRMRouter{rms: rms}

			rps, err := m.GetAgents()
			require.NoError(t, err)
			require.Equal(t, tt.managers*len(tt.agentNames), len(rps.Agents))

			for _, rp := range rps.Agents {
				require.Subset(t, tt.agentNames, rp.ResourcePools)
			}
		})
	}
}

func TestGetAgent(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	req := &apiv1.GetAgentRequest{AgentId: rp, ResourceManager: manager}
	res := &apiv1.GetAgentResponse{}

	mockRM.On("GetAgent", manager, req).Return(res, nil)

	ret, err := m.GetAgent(manager, req)
	require.NoError(t, err)
	require.Equal(t, ret, res)

	// Check that bogus RM call errors.
	ret, err = m.GetAgent("bogus", req)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
	require.Empty(t, ret)
}

func TestEnableAgent(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	req := &apiv1.EnableAgentRequest{AgentId: rp, ResourceManager: manager}
	res := &apiv1.EnableAgentResponse{}

	mockRM.On("EnableAgent", manager, req).Return(res, nil)

	ret, err := m.EnableAgent(manager, req)
	require.NoError(t, err)
	require.Equal(t, ret, res)

	// Check that bogus RM call errors.
	ret, err = m.EnableAgent("bogus", req)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
	require.Empty(t, ret, res)
}

func TestDisableAgent(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	req := &apiv1.DisableAgentRequest{AgentId: rp, ResourceManager: manager}
	res := &apiv1.DisableAgentResponse{}

	mockRM.On("DisableAgent", manager, req).Return(res, nil)

	ret, err := m.DisableAgent(manager, req)
	require.NoError(t, err)
	require.Equal(t, ret, res)

	// Check that bogus RM call errors.
	ret, err = m.DisableAgent("bogus", req)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
	require.Empty(t, ret)
}

func TestGetSlots(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	req := &apiv1.GetSlotsRequest{AgentId: rp, ResourceManager: manager}
	res := &apiv1.GetSlotsResponse{}

	mockRM.On("GetSlots", manager, req).Return(res, nil)

	ret, err := m.GetSlots(manager, req)
	require.NoError(t, err)
	require.Equal(t, ret, res)

	// Check that bogus RM call errors.
	ret, err = m.GetSlots("bogus", req)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
	require.Empty(t, ret)
}

func TestGetSlot(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	req := &apiv1.GetSlotRequest{AgentId: rp, ResourceManager: manager}
	res := &apiv1.GetSlotResponse{}

	mockRM.On("GetSlot", manager, req).Return(res, nil)

	ret, err := m.GetSlot(manager, req)
	require.NoError(t, err)
	require.Equal(t, ret, res)

	// Check that bogus RM call errors.
	ret, err = m.GetSlot("bogus", req)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
	require.Empty(t, ret)
}

func TestEnableSlot(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	req := &apiv1.EnableSlotRequest{AgentId: rp, ResourceManager: manager}
	res := &apiv1.EnableSlotResponse{}

	mockRM.On("EnableSlot", manager, req).Return(res, nil)

	ret, err := m.EnableSlot(manager, req)
	require.NoError(t, err)
	require.Equal(t, ret, res)

	// Check that bogus RM call errors.
	ret, err = m.EnableSlot("bogus", req)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
	require.Empty(t, ret)
}

func TestDisableSlot(t *testing.T) {
	mockRM := mocks.ResourceManager{}
	manager := uuid.NewString()
	log := logrus.WithField("component", "resource-router")
	m := &MultiRMRouter{
		defaultRMName: manager,
		rms:           map[string]rm.ResourceManager{manager: &mockRM},
		syslog:        log,
	}

	req := &apiv1.DisableSlotRequest{AgentId: rp, ResourceManager: manager}
	res := &apiv1.DisableSlotResponse{}

	mockRM.On("DisableSlot", manager, req).Return(res, nil)

	ret, err := m.DisableSlot(manager, req)
	require.NoError(t, err)
	require.Equal(t, ret, res)

	// Check that bogus RM call errors.
	ret, err = m.DisableSlot("bogus", req)
	require.Equal(t, err, ErrRMNotDefined("bogus"))
	require.Empty(t, ret)
}

func TestGetRMName(t *testing.T) {
	def := mocks.ResourceManager{}
	def.On("GetResourcePools").Return(&apiv1.GetResourcePoolsResponse{
		ResourcePools: []*resourcepoolv1.ResourcePool{
			{Name: "gcp2"},
		},
	}, nil)

	gcp := mocks.ResourceManager{}
	gcp.On("GetResourcePools").Return(&apiv1.GetResourcePoolsResponse{
		ResourcePools: []*resourcepoolv1.ResourcePool{
			{Name: "gcp1"}, {Name: "gcp2"},
		},
	}, nil)

	aws := mocks.ResourceManager{}
	aws.On("GetResourcePools").Return(&apiv1.GetResourcePoolsResponse{
		ResourcePools: []*resourcepoolv1.ResourcePool{
			{Name: "aws1"}, {Name: "gcp2"},
		},
	}, nil)

	mockMultiRM := MultiRMRouter{
		defaultRMName: "default",
		rms: map[string]rm.ResourceManager{
			"default": &def,
			"gcp":     &gcp,
			"aws":     &aws,
		},
		syslog: logrus.WithField("component", "resource-router"),
	}

	cases := []struct {
		name           string
		rmName         string
		rpName         string
		err            error
		expectedRMName string
		rmConflicts    []string
	}{
		{"RM/RP undefined", "", "", nil, mockMultiRM.defaultRMName, nil},
		{"RM defined, RP undefined", "aws", "", nil, "aws", nil},
		{"RM defined/doesn't exist, RP undefined", "aws123", "", ErrRMNotDefined("aws123"), "aws123", nil},
		{"RM defined, RP defined", "aws", "aws1", nil, "aws", nil},
		{"RM defined, RP defined/doesn't exist", "aws", "awsa", nil, "aws", nil},
		{"RM undefined, RP defined", "", "aws1", nil, "aws", nil},
		{
			"RM undefined, RP defined + conflict", "", "gcp2", ErrRMConflict([]string{"default", "gcp", "aws"}, "gcp2"),
			"",
			[]string{"default", "gcp", "aws"},
		},
		{"RM undefined, RP defined/doesn't exist", "", "gcp3", ErrRPNotDefined("gcp3"), "", nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rmName, err := mockMultiRM.getRM(tt.rmName, tt.rpName)
			require.Equal(t, tt.expectedRMName, rmName)
			if tt.err != nil && strings.Contains(tt.err.Error(), "exists for both resource managers") {
				require.ErrorContains(t, err, "exists for both resource managers")
				for _, r := range tt.rmConflicts {
					require.ErrorContains(t, err, r)
				}
			} else {
				require.Equal(t, tt.err, err)
			}
		})
	}
}
