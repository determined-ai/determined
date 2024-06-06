package multirm

import (
	"fmt"
	"os"
	"strings"
	"strconv"
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

const (
	additionalRMName = "additional"
	defaultRMName    = "default"
	emptyRPName      = rm.ResourcePoolName("")
)

var testMultiRM *MultiRMRouter

func TestMain(m *testing.M) {
	testMultiRM = &MultiRMRouter{
		defaultRMName: defaultRMName,
		rms: map[string]rm.ResourceManager{
			defaultRMName:   mockRM(rm.ResourcePoolName(defaultRMName)),
			"additional-rm": mockRM(rm.ResourcePoolName(additionalRMName)),
		},
		syslog: logrus.WithField("component", "resource-router"),
	}

	os.Exit(m.Run())
}

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
					a := alloc + strconv.Itoa(i)
					ret[*model.NewAllocationID(&a)] = sproto.AllocationSummary{}
				}
				require.Len(t, ret, len(tt.allocNames))

				mockRM := mocks.ResourceManager{}
				mockRM.On("GetAllocationSummaries").Return(ret, nil)

				rms[uuid.NewString()] = &mockRM
			}

			m := &MultiRMRouter{rms: rms}

			allocs, err := m.GetAllocationSummaries()
			require.NoError(t, err)
			require.Len(t, allocs, tt.managers*len(tt.allocNames))
			require.NotNil(t, allocs)

			bogus := "bogus"
			require.Empty(t, allocs[*model.NewAllocationID(&bogus)])

			for _, name := range tt.allocNames {
				n := name + "0"

				require.NotNil(t, allocs[*model.NewAllocationID(&n)])
				require.Empty(t, allocs[*model.NewAllocationID(&name)])
			}
		})
	}
}

func TestAllocate(t *testing.T) {
	cases := []struct {
		name string
		req  sproto.AllocateRequest
		res  *sproto.ResourcesSubscription
		err  error
	}{
		{"empty RP name will default", sproto.AllocateRequest{}, &sproto.ResourcesSubscription{}, nil},
		{
			"defined RP in default",
			sproto.AllocateRequest{ResourcePool: defaultRMName},
			&sproto.ResourcesSubscription{}, nil,
		},
		{
			"defined RP in additional RM",
			sproto.AllocateRequest{ResourcePool: additionalRMName},
			&sproto.ResourcesSubscription{}, nil,
		},
		{"undefined RP", sproto.AllocateRequest{ResourcePool: "bogus"}, nil, ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := testMultiRM.Allocate(tt.req)
			require.Equal(t, tt.err, err)
			require.Equal(t, tt.res, res)
		})
	}
}

func TestValidateResources(t *testing.T) {
	cases := []struct {
		name string
		req  sproto.ValidateResourcesRequest
		err  error
	}{
		{"empty RP name will default", sproto.ValidateResourcesRequest{}, nil},
		{"defined RP in default", sproto.ValidateResourcesRequest{ResourcePool: defaultRMName}, nil},
		{"defined RP in additional RM", sproto.ValidateResourcesRequest{ResourcePool: additionalRMName}, nil},
		{"undefined RP", sproto.ValidateResourcesRequest{ResourcePool: "bogus"}, ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testMultiRM.ValidateResources(tt.req)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestDeleteJob(t *testing.T) {
	cases := []struct {
		name string
		req  sproto.DeleteJob
		err  error
	}{
		{"MultiRM doesn't implement DeleteJob", sproto.DeleteJob{}, nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testMultiRM.DeleteJob(tt.req)
			require.NoError(t, err)
		})
	}
}

func TestNotifyContainerRunning(t *testing.T) {
	cases := []struct {
		name string
		req  sproto.NotifyContainerRunning
		err  error
	}{
		{"MultiRM doesn't implement NotifyContainerRunning", sproto.NotifyContainerRunning{}, rmerrors.ErrNotSupported},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := testMultiRM.NotifyContainerRunning(tt.req)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestSetGroupWeight(t *testing.T) {
	cases := []struct {
		name string
		req  sproto.SetGroupWeight
		err  error
	}{
		{"empty RP name will default", sproto.SetGroupWeight{}, nil},
		{"defined RP in default", sproto.SetGroupWeight{ResourcePool: defaultRMName}, nil},
		{"defined RP in additional RM", sproto.SetGroupWeight{ResourcePool: additionalRMName}, nil},
		{"undefined RP", sproto.SetGroupWeight{ResourcePool: "bogus"}, ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := testMultiRM.SetGroupWeight(tt.req)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestSetGroupPriority(t *testing.T) {
	cases := []struct {
		name string
		req  sproto.SetGroupPriority
		err  error
	}{
		{"empty RP name will default", sproto.SetGroupPriority{}, nil},
		{"defined RP in default", sproto.SetGroupPriority{ResourcePool: defaultRMName}, nil},
		{"defined RP in additional RM", sproto.SetGroupPriority{ResourcePool: additionalRMName}, nil},
		{"undefined RP", sproto.SetGroupPriority{ResourcePool: "bogus"}, ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := testMultiRM.SetGroupPriority(tt.req)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestExternalPreemptionPending(t *testing.T) {
	cases := []struct {
		name string
		req  sproto.PendingPreemption
		err  error
	}{
		{"MultiRM doesn't implement ExternalPreemptionPending", sproto.PendingPreemption{}, rmerrors.ErrNotSupported},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := testMultiRM.ExternalPreemptionPending(tt.req)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestIsReattachable(t *testing.T) {
	val := testMultiRM.IsReattachableOnlyAfterStarted()
	require.True(t, val)
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
			require.Len(t, rps.ResourcePools, tt.managers*len(tt.rpNames))

			for _, rp := range rps.ResourcePools {
				require.Contains(t, tt.rpNames, rp.Name)
			}
		})
	}
}

func TestGetDefaultResourcePools(t *testing.T) {
	cases := []struct {
		name string
		res  rm.ResourcePoolName
		err  error
	}{
		{"route to default pool", rm.ResourcePoolName("default"), nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := testMultiRM.GetDefaultComputeResourcePool()
			require.Equal(t, tt.err, err)
			require.Equal(t, tt.res, res)

			res, err = testMultiRM.GetDefaultAuxResourcePool()
			require.Equal(t, tt.err, err)
			require.Equal(t, tt.res, res)
		})
	}
}

func TestValidateResourcePool(t *testing.T) {
	cases := []struct {
		name   string
		rpName rm.ResourcePoolName
		err    error
	}{
		{"empty RP name will default", "", nil},
		{"defined RP in default", defaultRMName, nil},
		{"defined RP in additional RM", additionalRMName, nil},
		{"undefined RP", "bogus", ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := testMultiRM.ValidateResourcePool(tt.rpName)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestResolveResourcePool(t *testing.T) {
	cases := []struct {
		name   string
		rpName rm.ResourcePoolName
		err    error
	}{
		{"empty RP name will default", emptyRPName, nil},
		{"defined RP in default", defaultRMName, nil},
		{"defined RP in additional RM", additionalRMName, nil},
		{"undefined RP", "bogus", ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rpName, err := testMultiRM.ResolveResourcePool(tt.rpName, 0, 0)
			require.Equal(t, tt.rpName, rpName)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestTaskContainerDefaults(t *testing.T) {
	cases := []struct {
		name   string
		rpName rm.ResourcePoolName
		err    error
	}{
		{"empty RP name will default", emptyRPName, nil},
		{"defined RP in default", defaultRMName, nil},
		{"defined RP in additional RM", additionalRMName, nil},
		{"undefined RP", "bogus", ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testMultiRM.TaskContainerDefaults(tt.rpName, model.TaskContainerDefaultsConfig{})
			require.Equal(t, tt.err, err)
		})
	}
}

func TestGetJobQ(t *testing.T) {
	cases := []struct {
		name   string
		rpName rm.ResourcePoolName
		err    error
	}{
		{"empty RP name will default", emptyRPName, nil},
		{"defined RP in default", defaultRMName, nil},
		{"defined RP in additional RM", additionalRMName, nil},
		{"undefined RP", "bogus", ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testMultiRM.GetJobQ(tt.rpName)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestGetJobQueueStatsRequest(t *testing.T) {
	cases := []struct {
		name        string
		req         *apiv1.GetJobQueueStatsRequest
		err         error
		expectedLen int
	}{
		// Per the mocks set-up, no matter the pools in the request, return the max # of responses because of
		// the fan-out call to all RMs.
		{"empty request", &apiv1.GetJobQueueStatsRequest{}, nil, 2},
		{"empty RP name will default", &apiv1.GetJobQueueStatsRequest{ResourcePools: []string{""}}, nil, 2},
		{"defined RP in default", &apiv1.GetJobQueueStatsRequest{ResourcePools: []string{defaultRMName}}, nil, 2},
		{"defined RP in additional RM", &apiv1.GetJobQueueStatsRequest{ResourcePools: []string{additionalRMName}}, nil, 2},
		{"undefined RP", &apiv1.GetJobQueueStatsRequest{ResourcePools: []string{"bogus"}}, nil, 2},
		{
			"undefined RP + defined RP",
			&apiv1.GetJobQueueStatsRequest{ResourcePools: []string{"bogus", defaultRMName}},
			nil, 2,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := testMultiRM.GetJobQueueStatsRequest(tt.req)
			require.Equal(t, tt.err, err)
			require.Len(t, res.Results, tt.expectedLen)
		})
	}
}

func TestMoveJob(t *testing.T) {
	cases := []struct {
		name string
		req  sproto.MoveJob
		err  error
	}{
		{"empty RP name will default", sproto.MoveJob{ResourcePool: ""}, nil},
		{"defined RP in default", sproto.MoveJob{ResourcePool: defaultRMName}, nil},
		{"defined RP in additional RM", sproto.MoveJob{ResourcePool: additionalRMName}, nil},
		{"undefined RP", sproto.MoveJob{ResourcePool: "bogus"}, ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := testMultiRM.MoveJob(tt.req)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestGetExternalJobs(t *testing.T) {
	cases := []struct {
		name   string
		rpName rm.ResourcePoolName
		err    error
	}{
		{"empty RP name will default", "", nil},
		{"defined RP in default", defaultRMName, nil},
		{"defined RP in additional RM", additionalRMName, nil},
		{"undefined RP", "bogus", ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testMultiRM.GetExternalJobs(tt.rpName)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestHealthCheck(t *testing.T) {
	rmA := &mocks.ResourceManager{}
	rmA.On("HealthCheck").Return([]model.ResourceManagerHealth{
		{
			Name:   "a",
			Status: model.Healthy,
		},
	}).Once()

	rmB := &mocks.ResourceManager{}
	rmB.On("HealthCheck").Return([]model.ResourceManagerHealth{
		{
			Name:   "b",
			Status: model.Unhealthy,
		},
	})

	m := &MultiRMRouter{rms: map[string]rm.ResourceManager{
		"a": rmA,
		"b": rmB,
	}}
	require.ElementsMatch(t, []model.ResourceManagerHealth{
		{
			Name:   "a",
			Status: model.Healthy,
		},
		{
			Name:   "b",
			Status: model.Unhealthy,
		},
	}, m.HealthCheck())
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
			require.Len(t, rps.Agents, tt.managers*len(tt.agentNames))

			for _, rp := range rps.Agents {
				require.Subset(t, tt.agentNames, rp.ResourcePools)
			}
		})
	}
}

func TestGetAgent(t *testing.T) {
	cases := []struct {
		name string
		req  *apiv1.GetAgentRequest
		err  error
	}{
		{"empty RP name will default", &apiv1.GetAgentRequest{}, nil},
		{"defined RP in default", &apiv1.GetAgentRequest{AgentId: defaultRMName}, nil},
		{"defined RP in additional RM", &apiv1.GetAgentRequest{AgentId: additionalRMName}, nil},
		{"undefined RP", &apiv1.GetAgentRequest{AgentId: "bogus"}, ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testMultiRM.GetAgent(tt.req)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestEnableAgent(t *testing.T) {
	cases := []struct {
		name string
		req  *apiv1.EnableAgentRequest
		err  error
	}{
		{"empty RP name will default", &apiv1.EnableAgentRequest{}, nil},
		{"defined RP in default", &apiv1.EnableAgentRequest{AgentId: defaultRMName}, nil},
		{"defined RP in additional RM", &apiv1.EnableAgentRequest{AgentId: additionalRMName}, nil},
		{"undefined RP", &apiv1.EnableAgentRequest{AgentId: "bogus"}, ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testMultiRM.EnableAgent(tt.req)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestDisableAgent(t *testing.T) {
	cases := []struct {
		name string
		req  *apiv1.DisableAgentRequest
		err  error
	}{
		{"empty RP name will default", &apiv1.DisableAgentRequest{}, nil},
		{"defined RP in default", &apiv1.DisableAgentRequest{AgentId: defaultRMName}, nil},
		{"defined RP in additional RM", &apiv1.DisableAgentRequest{AgentId: additionalRMName}, nil},
		{"undefined RP", &apiv1.DisableAgentRequest{AgentId: "bogus"}, ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testMultiRM.DisableAgent(tt.req)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestGetSlots(t *testing.T) {
	cases := []struct {
		name string
		req  *apiv1.GetSlotsRequest
		err  error
	}{
		{"empty RP name will default", &apiv1.GetSlotsRequest{}, nil},
		{"defined RP in default", &apiv1.GetSlotsRequest{AgentId: defaultRMName}, nil},
		{"defined RP in additional RM", &apiv1.GetSlotsRequest{AgentId: additionalRMName}, nil},
		{"undefined RP", &apiv1.GetSlotsRequest{AgentId: "bogus"}, ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testMultiRM.GetSlots(tt.req)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestGetSlot(t *testing.T) {
	cases := []struct {
		name string
		req  *apiv1.GetSlotRequest
		err  error
	}{
		{"empty RP name will default", &apiv1.GetSlotRequest{}, nil},
		{"defined RP in default", &apiv1.GetSlotRequest{AgentId: defaultRMName}, nil},
		{"defined RP in additional RM", &apiv1.GetSlotRequest{AgentId: additionalRMName}, nil},
		{"undefined RP", &apiv1.GetSlotRequest{AgentId: "bogus"}, ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testMultiRM.GetSlot(tt.req)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestEnableSlot(t *testing.T) {
	cases := []struct {
		name string
		req  *apiv1.EnableSlotRequest
		err  error
	}{
		{"empty RP name will default", &apiv1.EnableSlotRequest{}, nil},
		{"defined RP in default", &apiv1.EnableSlotRequest{AgentId: defaultRMName}, nil},
		{"defined RP in additional RM", &apiv1.EnableSlotRequest{AgentId: additionalRMName}, nil},
		{"undefined RP", &apiv1.EnableSlotRequest{AgentId: "bogus"}, ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testMultiRM.EnableSlot(tt.req)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestDisableSlot(t *testing.T) {
	cases := []struct {
		name string
		req  *apiv1.DisableSlotRequest
		err  error
	}{
		{"empty RP name will default", &apiv1.DisableSlotRequest{}, nil},
		{"defined RP in default", &apiv1.DisableSlotRequest{AgentId: defaultRMName}, nil},
		{"defined RP in additional RM", &apiv1.DisableSlotRequest{AgentId: additionalRMName}, nil},
		{"undefined RP", &apiv1.DisableSlotRequest{AgentId: "bogus"}, ErrRPNotDefined("bogus")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testMultiRM.DisableSlot(tt.req)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestVerifyNamespaceExists(t *testing.T) {
	rm1 := mocks.ResourceManager{}
	rm1ClusterName := "cluster1"
	rm2ClusterName := "cluster2"
	rm2 := mocks.ResourceManager{}
	mockMultiRM := MultiRMRouter{
		rms: map[string]rm.ResourceManager{
			rm1ClusterName: &rm1,
			rm2ClusterName: &rm2,
		},
		syslog: logrus.WithField("component", "resource-router"),
	}

	validNamespaceName := "good-namespace"
	invalidNamespaceName := "bad-namespace"
	cases := []struct {
		name          string
		namespaceName string
		clusterName   string
		setupMockRM   func()
		err           error
	}{
		{
			"valid-namespace",
			validNamespaceName,
			rm1ClusterName,
			func() {
				rm1.On("VerifyNamespaceExists", validNamespaceName, rm1ClusterName).
					Return(nil).Once()
			},
			nil,
		},
		{
			"invalid-namespace-name",
			invalidNamespaceName,
			rm2ClusterName,
			func() {
				rm2.On("VerifyNamespaceExists", invalidNamespaceName, rm2ClusterName).
					Return(fmt.Errorf("namespace %s does not exist", invalidNamespaceName)).Once()
			},
			fmt.Errorf("namespace %s does not exist", invalidNamespaceName),
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			test.setupMockRM()
			err := mockMultiRM.VerifyNamespaceExists(test.namespaceName, test.clusterName)
			if err == nil {
				require.Equal(t, test.err, err)
			} else {
				require.True(t, strings.Contains(err.Error(), "does not exist"))
			}
		})
	}
}

func TestGetRMName(t *testing.T) {
	def := mocks.ResourceManager{}
	def.On("GetResourcePools").Return(&apiv1.GetResourcePoolsResponse{
		ResourcePools: []*resourcepoolv1.ResourcePool{
			{Name: "def1"},
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
			{Name: "aws1"}, {Name: "aws2"},
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
		rpName         rm.ResourcePoolName
		err            error
		expectedRMName string
	}{
		{"RP undefined, will default", "", nil, mockMultiRM.defaultRMName},
		{"RP defined in default", "def1", nil, "default"},
		{"RP defined in aws", "aws1", nil, "aws"},
		{"RP doesn't exist", "aws123", ErrRPNotDefined("aws123"), ""},
		{"RP doesn't exist", "gcp3", ErrRPNotDefined("gcp3"), ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rmName, err := mockMultiRM.getRMName(tt.rpName)
			require.Equal(t, tt.expectedRMName, rmName)
			require.Equal(t, tt.err, err)
		})
	}
}

func TestGetRM(t *testing.T) {
	defaultRM := &mocks.ResourceManager{}
	otherRM := &mocks.ResourceManager{}

	m := &MultiRMRouter{
		defaultRMName: defaultRMName,
		rms: map[string]rm.ResourceManager{
			defaultRMName: defaultRM,
			"rm1":         otherRM,
		},
	}

	cases := []struct {
		name          string
		rmClusterName string
		err           error
		expectedRM    *mocks.ResourceManager
	}{
		{
			"get-default-rm",
			"",
			nil,
			defaultRM,
		},
		{
			"get-existing-rm",
			"rm1",
			nil,
			otherRM,
		},
		{
			"get-nonexistent-rm",
			"badRM",
			rmerrors.ErrResourceManagerDNE,
			nil,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			rmToUse, err := m.getRM(test.rmClusterName)
			require.Equal(t, test.err, err)
			if test.expectedRM != nil {
				require.Equal(t, test.expectedRM, rmToUse)
			} else {
				require.Equal(t, nil, rmToUse)
			}
		})
	}
}

func mockRM(poolName rm.ResourcePoolName) *mocks.ResourceManager {
	mockRM := mocks.ResourceManager{}
	mockRM.On("GetResourcePools").Return(&apiv1.GetResourcePoolsResponse{
		ResourcePools: []*resourcepoolv1.ResourcePool{{Name: poolName.String()}},
	}, nil)
	mockRM.On("Allocate", mock.Anything).Return(&sproto.ResourcesSubscription{}, nil)
	mockRM.On("ValidateResources", mock.Anything).Return(nil, nil)
	mockRM.On("DeleteJob", mock.Anything).Return(sproto.EmptyDeleteJobResponse(), nil)
	mockRM.On("SetGroupWeight", mock.Anything).Return(nil)
	mockRM.On("SetGroupPriority", mock.Anything).Return(nil)
	mockRM.On("IsReattachableOnlyAfterStarted", mock.Anything).Return(true)
	mockRM.On("GetDefaultComputeResourcePool").Return(poolName, nil)
	mockRM.On("GetDefaultAuxResourcePool").Return(poolName, nil)
	mockRM.On("ValidateResourcePool", mock.Anything).Return(nil)

	mockRM.On("ResolveResourcePool", poolName, mock.Anything, mock.Anything).Return(poolName, nil)
	mockRM.On("ResolveResourcePool", emptyRPName, mock.Anything, mock.Anything).Return(emptyRPName, nil)

	mockRM.On("TaskContainerDefaults", mock.Anything, mock.Anything).Return(model.TaskContainerDefaultsConfig{}, nil)
	mockRM.On("GetJobQ", mock.Anything).Return(map[model.JobID]*sproto.RMJobInfo{}, nil)
	mockRM.On("GetJobQueueStatsRequest", mock.Anything).Return(&apiv1.GetJobQueueStatsResponse{
		Results: []*apiv1.RPQueueStat{{ResourcePool: poolName.String()}},
	}, nil)
	mockRM.On("MoveJob", mock.Anything).Return(nil)
	mockRM.On("GetExternalJobs", mock.Anything).Return([]*jobv1.Job{}, nil)
	mockRM.On("GetAgent", mock.Anything).Return(&apiv1.GetAgentResponse{}, nil)
	mockRM.On("EnableAgent", mock.Anything).Return(&apiv1.EnableAgentResponse{}, nil)
	mockRM.On("DisableAgent", mock.Anything).Return(&apiv1.DisableAgentResponse{}, nil)
	mockRM.On("GetSlots", mock.Anything).Return(&apiv1.GetSlotsResponse{}, nil)
	mockRM.On("GetSlot", mock.Anything).Return(&apiv1.GetSlotResponse{}, nil)
	mockRM.On("EnableSlot", mock.Anything).Return(&apiv1.EnableSlotResponse{}, nil)
	mockRM.On("DisableSlot", mock.Anything).Return(&apiv1.DisableSlotResponse{}, nil)
	mockRM.On("DefaultNamespace", mock.Anything).Return("default", nil)
	mockRM.On("VerifyNamespaceExists", mock.Anything).Return(nil)
	return &mockRM
}
