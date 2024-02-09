//go:build integration
// +build integration

package kubernetesrm

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	k8sV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/containerv1"
)

const (
	auxNode1Name        = "aux"
	auxNode2Name        = "aux2"
	compNode1Name       = "comp"
	compNode2Name       = "comp2"
	pod1NumSlots        = 4
	pod2NumSlots        = 8
	nodeNumSlots        = int64(8)
	nodeNumSlotsCPU     = int64(20)
	slotTypeGPU         = "randomDefault"
	cpuResourceRequests = int64(4000)
	nonDetNodeName      = "NonDetermined"
)

func TestGetAgents(t *testing.T) {
	type AgentsTestCase struct {
		Name           string
		podsService    *pods
		wantedAgentIDs map[string]int
	}

	auxNode1, auxNode2, compNode1, compNode2 := setupNodes()

	agentsTests := []AgentsTestCase{
		{
			Name: "GetAgents-CPU-NoPodLabels-NoAgents",
			podsService: createMockPodsService(make(map[string]*k8sV1.Node),
				device.CPU,
				false,
			),
			wantedAgentIDs: make(map[string]int),
		},
		{
			Name: "GetAgents-CPU-NoPodLabels",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				auxNode1Name: auxNode1,
				auxNode2Name: auxNode2,
			},
				device.CPU,
				false,
			),
			wantedAgentIDs: map[string]int{auxNode1Name: 0, auxNode2Name: 0},
		},
		{
			Name: "GetAgents-CPU-PodLabels",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				auxNode1Name: auxNode1,
				auxNode2Name: auxNode2,
			},
				device.CPU,
				true,
			),
			wantedAgentIDs: map[string]int{auxNode1Name: 0, auxNode2Name: 0, nonDetNodeName: 0},
		},
		{
			Name: "GetAgents-GPU-PodLabels-NonDetAgent",
			podsService: createMockPodsService(make(map[string]*k8sV1.Node),
				slotTypeGPU,
				true,
			),
			wantedAgentIDs: map[string]int{nonDetNodeName: 0},
		},
		{
			Name: "GetAgents-GPU-NoPodNoLabels",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				false,
			),
			wantedAgentIDs: map[string]int{compNode1Name: 0, compNode2Name: 0},
		},
		{
			Name: "GetAgents-GPU-PodLabels",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				true,
			),
			wantedAgentIDs: map[string]int{compNode1Name: 0, compNode2Name: 0, nonDetNodeName: 0},
		},
		{
			Name: "GetAgents-CUDA-NoPodLabels",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				false,
			),
			wantedAgentIDs: map[string]int{compNode1Name: 0, compNode2Name: 0},
		},
		{
			Name: "GetAgents-CUDA-PodLabels",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				true,
			),
			wantedAgentIDs: map[string]int{compNode1Name: 0, compNode2Name: 0, nonDetNodeName: 0},
		},
	}

	for _, test := range agentsTests {
		t.Run(test.Name, func(t *testing.T) {
			agentsResp := test.podsService.handleGetAgentsRequest()
			require.Equal(t, len(test.wantedAgentIDs), len(agentsResp.Agents))
			for _, agent := range agentsResp.Agents {
				_, ok := test.wantedAgentIDs[agent.Id]
				require.True(t, ok,
					fmt.Sprintf("name %s is not present in agent id list", agent.Id))
			}
		})
	}
}

func TestGetAgentsAllocationInfo(t *testing.T) {
	type AgentsTestCase struct {
		Name            string
		podsService     *pods
		wantedNumOfJobs int
	}

	auxNode1, auxNode2, compNode1, compNode2 := setupNodes()

	agentsTests := []AgentsTestCase{
		{
			Name: "GetAgents-CPU-NoPodLabels",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				auxNode1Name: auxNode1,
				auxNode2Name: auxNode2,
			},
				device.CPU,
				false,
			),
			wantedNumOfJobs: 1,
		},
		{
			Name: "GetAgents-CPU-PodLabels",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				auxNode1Name: auxNode1,
				auxNode2Name: auxNode2,
			},
				device.CPU,
				true,
			),
			wantedNumOfJobs: 1,
		},
		{
			Name: "GetAgents-GPU-NoPodLabels",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				false,
			),
			wantedNumOfJobs: 1,
		},
		{
			Name: "GetAgents-GPU-PodLabels",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				true,
			),
			wantedNumOfJobs: 1,
		},
		{
			Name: "GetAgents-CUDA-NoPodLabels",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				false,
			),
			wantedNumOfJobs: 1,
		},
		{
			Name: "GetAgents-CUDA-PodLabels",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				true,
			),
			wantedNumOfJobs: 1,
		},
	}

	for _, test := range agentsTests {
		t.Run(test.Name, func(t *testing.T) {
			wantedJobInfo := test.podsService.podNameToPodHandler

			agentsResp := test.podsService.handleGetAgentsRequest()
			jobInfo := make(map[string]*containerv1.Container)
			for _, agent := range agentsResp.Agents {
				for _, slot := range agent.Slots {
					if slot.Container != nil {
						job, ok := wantedJobInfo[slot.Container.AllocationId]
						require.True(t, ok,
							fmt.Sprintf("allocation id %s is unexpected", slot.Container.AllocationId))

						wantedJobID := job.jobID
						wantedTaskID := model.AllocationID(slot.Container.AllocationId).ToTaskID()
						jobID := model.JobID(slot.Container.JobId)
						taskID := model.TaskID(slot.Container.TaskId)
						require.Equal(t, wantedJobID, jobID,
							fmt.Sprintf("expected: %v, got %v", wantedJobID, jobID))
						require.Equal(t, wantedTaskID, taskID,
							fmt.Sprintf("expected: %v, got %v", wantedTaskID, taskID))

						jobInfo[slot.Container.AllocationId] = slot.Container
					}
				}
			}

			require.Equal(t, test.wantedNumOfJobs, len(jobInfo))
		})
	}
}

func TestGetAgent(t *testing.T) {
	type AgentTestCase struct {
		Name          string
		podsService   *pods
		agentExists   bool
		wantedAgentID string
	}

	auxNode1, auxNode2, compNode1, compNode2 := setupNodes()

	agentTests := []AgentTestCase{
		{
			Name: "GetAgent-CPU-NoPodLabels-Aux1",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				auxNode1Name: auxNode1,
				auxNode2Name: auxNode2,
			},
				device.CPU,
				false,
			),
			agentExists:   true,
			wantedAgentID: auxNode1Name,
		},
		{
			Name: "GetAgent-CPU-PodLabels-Aux2",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				auxNode1Name: auxNode1,
				auxNode2Name: auxNode2,
			},
				device.CPU,
				true,
			),
			agentExists:   true,
			wantedAgentID: auxNode2Name,
		},
		{
			Name: "GetAgent-GPU-PodLabels-Comp1",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				true,
			),
			agentExists:   true,
			wantedAgentID: compNode1Name,
		},
		{
			Name: "GetAgent-CUDA-NoPodLabels-Comp2",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				false,
			),
			agentExists:   true,
			wantedAgentID: compNode2Name,
		},
		{
			Name: "GetAgent-CUDA-NoPodLabels-NonexistentAgent",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				false,
			),
			agentExists:   false,
			wantedAgentID: uuid.NewString(),
		},
		{
			Name: "GetAgent-CUDA-NoPodLabels-EmptyAgentID",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				false,
			),
			agentExists:   false,
			wantedAgentID: "",
		},
	}

	for _, test := range agentTests {
		t.Run(test.Name, func(t *testing.T) {
			agentResp := test.podsService.handleGetAgentRequest(test.wantedAgentID)
			if agentResp == nil {
				require.True(t, !test.agentExists)
				return
			}
			require.Equal(t, test.wantedAgentID, agentResp.Agent.Id)
		})
	}
}

func TestGetSlots(t *testing.T) {
	type SlotsTestCase struct {
		Name           string
		podsService    *pods
		agentID        string
		agentExists    bool
		wantedSlotsNum int
	}

	auxNode1, auxNode2, compNode1, compNode2 := setupNodes()

	slotsTests := []SlotsTestCase{
		{
			Name: "GetSlots-CPU-NoPodLabels-Aux1",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				auxNode1Name: auxNode1,
				auxNode2Name: auxNode2,
			},
				device.CPU,
				false,
			),
			agentID:        auxNode1Name,
			agentExists:    true,
			wantedSlotsNum: int(nodeNumSlots),
		},
		{
			Name: "GetSlots-GPU-NoPodLabels-Comp2",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				false,
			),
			agentID:        compNode2Name,
			agentExists:    true,
			wantedSlotsNum: 8,
		},
		{
			Name: "GetSlots-CUDA-PodLabels-Comp1",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				true,
			),
			agentID:        compNode1Name,
			agentExists:    true,
			wantedSlotsNum: int(nodeNumSlots),
		},
		{
			Name: "GetSlots-CUDA-PodLabels-NonexistentAgent",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				true,
			),
			agentID:        uuid.NewString(),
			agentExists:    false,
			wantedSlotsNum: 0,
		},
		{
			Name: "GetSlots-CUDA-PodLabels-EmptyAgentID",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				true,
			),
			agentID:        "",
			agentExists:    false,
			wantedSlotsNum: 0,
		},
	}

	// Number of active slots on given nodes.
	nodeToSlots := map[string]int{
		auxNode1Name:  pod1NumSlots,
		compNode1Name: pod2NumSlots,
		auxNode2Name:  0,
		compNode2Name: 0,
	}
	for _, test := range slotsTests {
		t.Run(test.Name, func(t *testing.T) {
			slotsResp := test.podsService.handleGetSlotsRequest(test.agentID)
			if slotsResp == nil {
				require.True(t, !test.agentExists)
				return
			}
			require.Equal(t, test.wantedSlotsNum, len(slotsResp.Slots))

			// Count number of active slots on the node. (Slots allocated to a pod running
			// a container).
			activeSlots := 0
			for _, slot := range slotsResp.Slots {
				slotID, err := strconv.Atoi(slot.Id)
				require.NoError(t, err)
				require.True(t, slotID >= 0 && slotID < int(nodeNumSlots),
					fmt.Sprintf("slot %s is out of range", slot.Id))
				if slot.Container != nil {
					activeSlots++
				}
			}
			require.Equal(t, nodeToSlots[test.agentID], activeSlots)
		})
	}
}

func TestGetSlot(t *testing.T) {
	type SlotTestCase struct {
		Name          string
		podsService   *pods
		agentID       string
		wantedSlotNum string
	}

	auxNode1, auxNode2, compNode1, compNode2 := setupNodes()

	slotTests := []SlotTestCase{
		{
			Name: "GetSlot-CPU-PodLabels-Aux1-LastId",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				auxNode1Name: auxNode1,
				auxNode2Name: auxNode2,
			},
				device.CPU,
				true,
			),
			agentID:       auxNode1Name,
			wantedSlotNum: strconv.Itoa(7),
		},
		{
			Name: "GetSlot-GPU-PodLabels-Comp1-Id4",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				true,
			),
			agentID:       compNode1Name,
			wantedSlotNum: strconv.Itoa(4),
		},
		{
			Name: "GetSlot-GPU-PodLabels-Comp1-Id0",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				true,
			),
			agentID:       compNode1Name,
			wantedSlotNum: strconv.Itoa(0),
		},
		{
			Name: "GetSlot-CUDA-NoPodLabels-Comp1-BadSlotReq",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				false,
			),
			agentID:       compNode1Name,
			wantedSlotNum: strconv.Itoa(-1),
		},
		{
			Name: "GetSlot-CUDA-PodLabels-Comp2-BadSlotReq",
			podsService: createMockPodsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				true,
			),
			agentID:       compNode2Name,
			wantedSlotNum: strconv.Itoa(0),
		},
	}

	for _, test := range slotTests {
		t.Run(test.Name, func(t *testing.T) {
			slotResp := test.podsService.handleGetSlotRequest(test.agentID, test.wantedSlotNum)
			if slotResp == nil {
				wantedSlotInt, err := strconv.Atoi(test.wantedSlotNum)
				require.NoError(t, err)
				require.True(t, wantedSlotInt < 0 || wantedSlotInt >= int(nodeNumSlots))
				return
			}
			require.Equal(t, test.wantedSlotNum, slotResp.Slot.Id)
		})
	}
}

func TestROCmPodsService(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func()
	}{
		{name: "GetAgentsROCM", testFunc: testROCMGetAgents},
		{name: "GetAgentROCM", testFunc: testROCMGetAgent},
		{name: "GetSlotsROCM", testFunc: testROCMGetSlots},
		{name: "GetSlotROCM", testFunc: testROCMGetSlot},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { require.Panics(t, test.testFunc) })
	}
}

func testROCMGetAgents() {
	ps := createMockPodsService(createCompNodeMap(), device.ROCM, false)
	ps.handleGetAgentsRequest()
}

func testROCMGetAgent() {
	nodes := createCompNodeMap()
	ps := createMockPodsService(nodes, device.ROCM, false)
	ps.handleGetAgentRequest(compNode1Name)
}

func testROCMGetSlots() {
	nodes := createCompNodeMap()
	ps := createMockPodsService(nodes, device.ROCM, false)
	ps.handleGetSlotsRequest(compNode1Name)
}

func testROCMGetSlot() {
	nodes := createCompNodeMap()
	ps := createMockPodsService(nodes, device.ROCM, false)
	for i := 0; i < int(nodeNumSlots); i++ {
		ps.handleGetSlotRequest(compNode1Name, strconv.Itoa(i))
	}
}

func setupNodes() (*k8sV1.Node, *k8sV1.Node, *k8sV1.Node, *k8sV1.Node) {
	auxResourceList := map[k8sV1.ResourceName]resource.Quantity{
		k8sV1.ResourceCPU: *resource.NewQuantity(nodeNumSlotsCPU, resource.DecimalSI),
	}

	compResourceList := map[k8sV1.ResourceName]resource.Quantity{
		k8sV1.ResourceName(ResourceTypeNvidia): *resource.NewQuantity(
			nodeNumSlots,
			resource.DecimalSI,
		),
	}

	auxNode1 := k8sV1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			ResourceVersion: "1",
			Name:            auxNode1Name,
		},
		Status: k8sV1.NodeStatus{Allocatable: auxResourceList},
	}

	auxNode2 := k8sV1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			ResourceVersion: "1",
			Name:            auxNode2Name,
		},
		Status: k8sV1.NodeStatus{Allocatable: auxResourceList},
	}

	compNode1 := k8sV1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			ResourceVersion: "1",
			Name:            compNode1Name,
		},
		Status: k8sV1.NodeStatus{Allocatable: compResourceList},
	}

	compNode2 := k8sV1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			ResourceVersion: "1",
			Name:            compNode2Name,
		},
		Status: k8sV1.NodeStatus{Allocatable: compResourceList},
	}
	return &auxNode1, &auxNode2, &compNode1, &compNode2
}

func createCompNodeMap() map[string]*k8sV1.Node {
	resourceList := map[k8sV1.ResourceName]resource.Quantity{
		k8sV1.ResourceName(device.ROCM): *resource.NewQuantity(nodeNumSlots, resource.DecimalSI),
	}

	compNode := k8sV1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			ResourceVersion: "1",
			Name:            compNode1Name,
		},
		Status: k8sV1.NodeStatus{Allocatable: resourceList},
	}
	return map[string]*k8sV1.Node{
		compNode.Name: &compNode,
	}
}

// createMockPodsService creates two pods. One pod is run on the auxiliary node and the other is
// run on the compute node.
func createMockPodsService(nodes map[string]*k8sV1.Node, devSlotType device.Type,
	labels bool,
) *pods {
	// Create two pods that are scheduled on a node.
	pod1 := &pod{
		allocationID: model.AllocationID(uuid.New().String() + ".1"),
		slots:        pod1NumSlots,
		pod: &k8sV1.Pod{
			Spec: k8sV1.PodSpec{NodeName: auxNode1Name},
		},
		jobID: model.JobID(uuid.New().String()),
	}
	pod2 := &pod{
		allocationID: model.AllocationID(uuid.New().String() + ".1"),
		slots:        pod2NumSlots,
		pod: &k8sV1.Pod{
			Spec: k8sV1.PodSpec{NodeName: compNode1Name},
		},
		jobID: model.JobID(uuid.New().String()),
	}

	// Create pod that is not yet scheduled on a node.
	pod3 := &pod{
		allocationID: model.AllocationID(uuid.New().String() + ".1"),
		slots:        0,
		pod: &k8sV1.Pod{
			Spec: k8sV1.PodSpec{NodeName: ""},
		},
		jobID: model.JobID(uuid.New().String()),
	}

	podsList := &k8sV1.PodList{Items: []k8sV1.Pod{*pod1.pod, *pod2.pod, *pod3.pod}}

	var nonDetPod *pod

	if labels {
		// Give labels to all determined pods.
		pod1.pod.ObjectMeta = metaV1.ObjectMeta{Labels: map[string]string{"determined": ""}}
		pod2.pod.ObjectMeta = metaV1.ObjectMeta{Labels: map[string]string{"determined": ""}}
		pod3.pod.ObjectMeta = metaV1.ObjectMeta{Labels: map[string]string{"determined": ""}}

		resourceList := make(map[k8sV1.ResourceName]resource.Quantity)

		if devSlotType == device.CPU {
			resourceList[k8sV1.ResourceName(device.CPU)] = *resource.NewQuantity(nodeNumSlotsCPU,
				resource.DecimalSI)
		} else {
			resourceList[k8sV1.ResourceName(ResourceTypeNvidia)] = *resource.NewQuantity(nodeNumSlots,
				resource.DecimalSI)
		}
		nonDetNode := k8sV1.Node{
			ObjectMeta: metaV1.ObjectMeta{
				ResourceVersion: "1",
				Name:            nonDetNodeName,
			},
			Status: k8sV1.NodeStatus{Allocatable: resourceList},
		}

		nodes[nonDetNode.Name] = &nonDetNode

		// Create pod without determined label.
		nonDetPod = &pod{
			allocationID: model.AllocationID(uuid.New().String() + ".1"),
			slots:        0,
			pod: &k8sV1.Pod{
				Spec: k8sV1.PodSpec{NodeName: nonDetNodeName},
			},
		}
		podsList.Items = append(podsList.Items, *nonDetPod.pod)
	}

	podHandlers := map[string]*pod{
		string(pod1.allocationID): pod1,
		string(pod2.allocationID): pod2,
		string(pod3.allocationID): pod3,
	}
	if nonDetPod != nil {
		podHandlers[string(nonDetPod.allocationID)] = nonDetPod
	}

	// Create pod service client set.
	podsClientSet := &mocks.K8sClientsetInterface{}
	coreV1Interface := &mocks.K8sCoreV1Interface{}
	podsInterface := &mocks.PodInterface{}
	podsInterface.On("List", mock.Anything, mock.Anything).Return(podsList, nil)
	coreV1Interface.On("Pods", mock.Anything).Return(podsInterface)
	podsClientSet.On("CoreV1").Return(coreV1Interface)

	return &pods{
		namespace:           "default",
		namespaceToPoolName: make(map[string]string),
		currentNodes:        nodes,
		podNameToPodHandler: podHandlers,
		slotType:            devSlotType,
		syslog:              logrus.WithField("namespace", namespace),
		nodeToSystemResourceRequests: map[string]int64{
			auxNode1Name: cpuResourceRequests,
			auxNode2Name: cpuResourceRequests,
		},
		slotResourceRequests: config.PodSlotResourceRequests{CPU: 2},
		clientSet:            podsClientSet,
	}
}
