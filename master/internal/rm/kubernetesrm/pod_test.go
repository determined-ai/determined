package kubernetesrm

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"gopkg.in/inf.v0"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/master/pkg/tasks"

	k8sV1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClient "k8s.io/client-go/kubernetes"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type mockReceiver struct {
	name      string
	responses []actor.Message
}

func newMockReceiver(name string) *mockReceiver {
	return &mockReceiver{name: name, responses: []actor.Message{}}
}

func (m *mockReceiver) Receive(ctx *actor.Context) error {
	m.responses = append(m.responses, ctx.Message())
	return nil
}

func (m *mockReceiver) GetLength() int {
	return len(m.responses)
}

func (m *mockReceiver) Purge() {
	m.responses = []actor.Message{}
}

func (m *mockReceiver) Pop() (actor.Message, error) {
	if len(m.responses) > 0 {
		output := m.responses[0]
		m.responses = m.responses[1:]
		return output, nil
	}
	return nil, fmt.Errorf("nothing left in responses")
}

func createPod(
	taskHandler *actor.Ref,
	clusterHandler *actor.Ref,
	resourceHandler *requestQueue,
	task tasks.TaskSpec,
) *pod {
	msg := StartTaskPod{
		TaskActor: taskHandler,
		Spec:      task,
		Slots:     1,
	}
	clusterID := "test"
	clientSet := k8sClient.Clientset{}
	namespace := "default"
	masterIP := "0.0.0.0"
	var masterPort int32 = 32
	podInterface := &mocks.PodInterface{}
	configMapInterface := clientSet.CoreV1().ConfigMaps(namespace)
	resourceRequestQueue := resourceHandler
	leaveKubernetesResources := false
	slotType := device.CUDA
	slotResourceRequests := config.PodSlotResourceRequests{}

	newPodHandler := newPod(
		msg, clusterID, &clientSet, namespace, masterIP, masterPort,
		model.TLSClientConfig{}, model.TLSClientConfig{},
		model.LoggingConfig{DefaultLoggingConfig: &model.DefaultLoggingConfig{}},
		podInterface, configMapInterface, resourceRequestQueue, leaveKubernetesResources,
		slotType, slotResourceRequests, "default-scheduler", config.DefaultFluentConfig,
	)

	return newPodHandler
}

func createReceivers(system *actor.System) (map[string]*mockReceiver, map[string]*actor.Ref) {
	podMap := make(map[string]*mockReceiver)
	actorMap := make(map[string]*actor.Ref)

	podMap["task"] = newMockReceiver("task-receiver")
	actorMap["task"], _ = system.ActorOf(
		actor.Addr("task-pod"),
		podMap["task"],
	)

	podMap["cluster"] = newMockReceiver("cluster-receiver")
	actorMap["cluster"], _ = system.ActorOf(
		actor.Addr("cluster-pod"),
		podMap["cluster"],
	)

	podMap["resource"] = newMockReceiver("resource-receiver")
	actorMap["resource"], _ = system.ActorOf(
		actor.Addr("resource-pod"),
		podMap["resource"],
	)

	return podMap, actorMap
}

func createAgentUserGroup() *model.AgentUserGroup {
	return &model.AgentUserGroup{
		ID:     1,
		UserID: 1,
		User:   "determined",
		UID:    1,
		Group:  "test-group",
		GID:    1,
	}
}

func createUser() *model.User {
	return &model.User{
		ID:       1,
		Username: "determined",
		Active:   true,
		Admin:    false,
	}
}

func createPodWithMockQueue() (
	*actor.System,
	*pod,
	*actor.Ref,
	map[string]*mockReceiver,
	map[string]*actor.Ref,
	map[string]*k8sV1.Pod,
) {
	commandSpec := tasks.GenericCommandSpec{
		Base: tasks.TaskSpec{
			AllocationID:     "task",
			ContainerID:      "container",
			ClusterID:        "cluster",
			AgentUserGroup:   createAgentUserGroup(),
			Owner:            createUser(),
			UserSessionToken: "bogus",
		},
		Config: model.CommandConfig{Description: "test-config"},
	}
	system := actor.NewSystem("test-sys")
	podMap, actorMap := createReceivers(system)
	pods := make(map[string]*k8sV1.Pod)
	podInterface := &mocks.PodInterface{}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}
	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	newPod := createPod(
		actorMap["task"],
		actorMap["cluster"],
		k8sRequestQueue,
		commandSpec.ToTaskSpec(),
	)

	pod := &k8sV1.Pod{ObjectMeta: metaV1.ObjectMeta{Name: newPod.podName,
		Labels:    map[string]string{"determined": "task"},
		Namespace: "default"},
		Spec: v1.PodSpec{Volumes: []v1.Volume{v1.Volume{Name: "det-shm-volume",
			VolumeSource: v1.VolumeSource{HostPath: (*v1.HostPathVolumeSource)(nil),
				EmptyDir:             (*v1.EmptyDirVolumeSource)(0x140009540a8),
				GCEPersistentDisk:    (*v1.GCEPersistentDiskVolumeSource)(nil),
				AWSElasticBlockStore: (*v1.AWSElasticBlockStoreVolumeSource)(nil),
				GitRepo:              (*v1.GitRepoVolumeSource)(nil),
				Secret:               (*v1.SecretVolumeSource)(nil),
				NFS:                  (*v1.NFSVolumeSource)(nil),
				ISCSI:                (*v1.ISCSIVolumeSource)(nil), Glusterfs: (*v1.GlusterfsVolumeSource)(nil), PersistentVolumeClaim: (*v1.PersistentVolumeClaimVolumeSource)(nil), RBD: (*v1.RBDVolumeSource)(nil), FlexVolume: (*v1.FlexVolumeSource)(nil), Cinder: (*v1.CinderVolumeSource)(nil), CephFS: (*v1.CephFSVolumeSource)(nil), Flocker: (*v1.FlockerVolumeSource)(nil), DownwardAPI: (*v1.DownwardAPIVolumeSource)(nil), FC: (*v1.FCVolumeSource)(nil), AzureFile: (*v1.AzureFileVolumeSource)(nil), ConfigMap: (*v1.ConfigMapVolumeSource)(nil), VsphereVolume: (*v1.VsphereVirtualDiskVolumeSource)(nil), Quobyte: (*v1.QuobyteVolumeSource)(nil), AzureDisk: (*v1.AzureDiskVolumeSource)(nil), PhotonPersistentDisk: (*v1.PhotonPersistentDiskVolumeSource)(nil), Projected: (*v1.ProjectedVolumeSource)(nil), PortworxVolume: (*v1.PortworxVolumeSource)(nil), ScaleIO: (*v1.ScaleIOVolumeSource)(nil), StorageOS: (*v1.StorageOSVolumeSource)(nil), CSI: (*v1.CSIVolumeSource)(nil), Ephemeral: (*v1.EphemeralVolumeSource)(nil)}}, v1.Volume{Name: "archive-volume", VolumeSource: v1.VolumeSource{HostPath: (*v1.HostPathVolumeSource)(nil), EmptyDir: (*v1.EmptyDirVolumeSource)(nil), GCEPersistentDisk: (*v1.GCEPersistentDiskVolumeSource)(nil), AWSElasticBlockStore: (*v1.AWSElasticBlockStoreVolumeSource)(nil), GitRepo: (*v1.GitRepoVolumeSource)(nil), Secret: (*v1.SecretVolumeSource)(nil), NFS: (*v1.NFSVolumeSource)(nil), ISCSI: (*v1.ISCSIVolumeSource)(nil), Glusterfs: (*v1.GlusterfsVolumeSource)(nil), PersistentVolumeClaim: (*v1.PersistentVolumeClaimVolumeSource)(nil), RBD: (*v1.RBDVolumeSource)(nil), FlexVolume: (*v1.FlexVolumeSource)(nil), Cinder: (*v1.CinderVolumeSource)(nil), CephFS: (*v1.CephFSVolumeSource)(nil), Flocker: (*v1.FlockerVolumeSource)(nil), DownwardAPI: (*v1.DownwardAPIVolumeSource)(nil), FC: (*v1.FCVolumeSource)(nil), AzureFile: (*v1.AzureFileVolumeSource)(nil), ConfigMap: (*v1.ConfigMapVolumeSource)(0x14000612180), VsphereVolume: (*v1.VsphereVirtualDiskVolumeSource)(nil), Quobyte: (*v1.QuobyteVolumeSource)(nil), AzureDisk: (*v1.AzureDiskVolumeSource)(nil), PhotonPersistentDisk: (*v1.PhotonPersistentDiskVolumeSource)(nil), Projected: (*v1.ProjectedVolumeSource)(nil), PortworxVolume: (*v1.PortworxVolumeSource)(nil), ScaleIO: (*v1.ScaleIOVolumeSource)(nil), StorageOS: (*v1.StorageOSVolumeSource)(nil), CSI: (*v1.CSIVolumeSource)(nil), Ephemeral: (*v1.EphemeralVolumeSource)(nil)}}, v1.Volume{Name: "entrypoint-volume", VolumeSource: v1.VolumeSource{HostPath: (*v1.HostPathVolumeSource)(nil), EmptyDir: (*v1.EmptyDirVolumeSource)(nil), GCEPersistentDisk: (*v1.GCEPersistentDiskVolumeSource)(nil), AWSElasticBlockStore: (*v1.AWSElasticBlockStoreVolumeSource)(nil), GitRepo: (*v1.GitRepoVolumeSource)(nil), Secret: (*v1.SecretVolumeSource)(nil), NFS: (*v1.NFSVolumeSource)(nil), ISCSI: (*v1.ISCSIVolumeSource)(nil), Glusterfs: (*v1.GlusterfsVolumeSource)(nil), PersistentVolumeClaim: (*v1.PersistentVolumeClaimVolumeSource)(nil), RBD: (*v1.RBDVolumeSource)(nil), FlexVolume: (*v1.FlexVolumeSource)(nil), Cinder: (*v1.CinderVolumeSource)(nil), CephFS: (*v1.CephFSVolumeSource)(nil), Flocker: (*v1.FlockerVolumeSource)(nil), DownwardAPI: (*v1.DownwardAPIVolumeSource)(nil), FC: (*v1.FCVolumeSource)(nil), AzureFile: (*v1.AzureFileVolumeSource)(nil), ConfigMap: (*v1.ConfigMapVolumeSource)(0x14000612200), VsphereVolume: (*v1.VsphereVirtualDiskVolumeSource)(nil), Quobyte: (*v1.QuobyteVolumeSource)(nil), AzureDisk: (*v1.AzureDiskVolumeSource)(nil), PhotonPersistentDisk: (*v1.PhotonPersistentDiskVolumeSource)(nil), Projected: (*v1.ProjectedVolumeSource)(nil), PortworxVolume: (*v1.PortworxVolumeSource)(nil), ScaleIO: (*v1.ScaleIOVolumeSource)(nil), StorageOS: (*v1.StorageOSVolumeSource)(nil), CSI: (*v1.CSIVolumeSource)(nil), Ephemeral: (*v1.EphemeralVolumeSource)(nil)}}, v1.Volume{Name: "additional-files-volume", VolumeSource: v1.VolumeSource{HostPath: (*v1.HostPathVolumeSource)(nil), EmptyDir: (*v1.EmptyDirVolumeSource)(0x140009540c0), GCEPersistentDisk: (*v1.GCEPersistentDiskVolumeSource)(nil), AWSElasticBlockStore: (*v1.AWSElasticBlockStoreVolumeSource)(nil), GitRepo: (*v1.GitRepoVolumeSource)(nil), Secret: (*v1.SecretVolumeSource)(nil), NFS: (*v1.NFSVolumeSource)(nil), ISCSI: (*v1.ISCSIVolumeSource)(nil), Glusterfs: (*v1.GlusterfsVolumeSource)(nil), PersistentVolumeClaim: (*v1.PersistentVolumeClaimVolumeSource)(nil), RBD: (*v1.RBDVolumeSource)(nil), FlexVolume: (*v1.FlexVolumeSource)(nil), Cinder: (*v1.CinderVolumeSource)(nil), CephFS: (*v1.CephFSVolumeSource)(nil), Flocker: (*v1.FlockerVolumeSource)(nil), DownwardAPI: (*v1.DownwardAPIVolumeSource)(nil), FC: (*v1.FCVolumeSource)(nil), AzureFile: (*v1.AzureFileVolumeSource)(nil), ConfigMap: (*v1.ConfigMapVolumeSource)(nil), VsphereVolume: (*v1.VsphereVirtualDiskVolumeSource)(nil), Quobyte: (*v1.QuobyteVolumeSource)(nil), AzureDisk: (*v1.AzureDiskVolumeSource)(nil), PhotonPersistentDisk: (*v1.PhotonPersistentDiskVolumeSource)(nil), Projected: (*v1.ProjectedVolumeSource)(nil), PortworxVolume: (*v1.PortworxVolumeSource)(nil), ScaleIO: (*v1.ScaleIOVolumeSource)(nil), StorageOS: (*v1.StorageOSVolumeSource)(nil), CSI: (*v1.CSIVolumeSource)(nil), Ephemeral: (*v1.EphemeralVolumeSource)(nil)}}, v1.Volume{Name: "det-logs", VolumeSource: v1.VolumeSource{HostPath: (*v1.HostPathVolumeSource)(nil), EmptyDir: (*v1.EmptyDirVolumeSource)(0x140009540d8), GCEPersistentDisk: (*v1.GCEPersistentDiskVolumeSource)(nil), AWSElasticBlockStore: (*v1.AWSElasticBlockStoreVolumeSource)(nil), GitRepo: (*v1.GitRepoVolumeSource)(nil), Secret: (*v1.SecretVolumeSource)(nil), NFS: (*v1.NFSVolumeSource)(nil), ISCSI: (*v1.ISCSIVolumeSource)(nil), Glusterfs: (*v1.GlusterfsVolumeSource)(nil), PersistentVolumeClaim: (*v1.PersistentVolumeClaimVolumeSource)(nil), RBD: (*v1.RBDVolumeSource)(nil), FlexVolume: (*v1.FlexVolumeSource)(nil), Cinder: (*v1.CinderVolumeSource)(nil), CephFS: (*v1.CephFSVolumeSource)(nil), Flocker: (*v1.FlockerVolumeSource)(nil), DownwardAPI: (*v1.DownwardAPIVolumeSource)(nil), FC: (*v1.FCVolumeSource)(nil), AzureFile: (*v1.AzureFileVolumeSource)(nil), ConfigMap: (*v1.ConfigMapVolumeSource)(nil), VsphereVolume: (*v1.VsphereVirtualDiskVolumeSource)(nil), Quobyte: (*v1.QuobyteVolumeSource)(nil), AzureDisk: (*v1.AzureDiskVolumeSource)(nil), PhotonPersistentDisk: (*v1.PhotonPersistentDiskVolumeSource)(nil), Projected: (*v1.ProjectedVolumeSource)(nil), PortworxVolume: (*v1.PortworxVolumeSource)(nil), ScaleIO: (*v1.ScaleIOVolumeSource)(nil), StorageOS: (*v1.StorageOSVolumeSource)(nil), CSI: (*v1.CSIVolumeSource)(nil), Ephemeral: (*v1.EphemeralVolumeSource)(nil)}}, v1.Volume{Name: "det-fluent", VolumeSource: v1.VolumeSource{HostPath: (*v1.HostPathVolumeSource)(nil), EmptyDir: (*v1.EmptyDirVolumeSource)(nil), GCEPersistentDisk: (*v1.GCEPersistentDiskVolumeSource)(nil), AWSElasticBlockStore: (*v1.AWSElasticBlockStoreVolumeSource)(nil), GitRepo: (*v1.GitRepoVolumeSource)(nil), Secret: (*v1.SecretVolumeSource)(nil), NFS: (*v1.NFSVolumeSource)(nil), ISCSI: (*v1.ISCSIVolumeSource)(nil), Glusterfs: (*v1.GlusterfsVolumeSource)(nil), PersistentVolumeClaim: (*v1.PersistentVolumeClaimVolumeSource)(nil), RBD: (*v1.RBDVolumeSource)(nil), FlexVolume: (*v1.FlexVolumeSource)(nil), Cinder: (*v1.CinderVolumeSource)(nil), CephFS: (*v1.CephFSVolumeSource)(nil), Flocker: (*v1.FlockerVolumeSource)(nil), DownwardAPI: (*v1.DownwardAPIVolumeSource)(nil), FC: (*v1.FCVolumeSource)(nil), AzureFile: (*v1.AzureFileVolumeSource)(nil), ConfigMap: (*v1.ConfigMapVolumeSource)(0x14000612600), VsphereVolume: (*v1.VsphereVirtualDiskVolumeSource)(nil), Quobyte: (*v1.QuobyteVolumeSource)(nil), AzureDisk: (*v1.AzureDiskVolumeSource)(nil), PhotonPersistentDisk: (*v1.PhotonPersistentDiskVolumeSource)(nil), Projected: (*v1.ProjectedVolumeSource)(nil), PortworxVolume: (*v1.PortworxVolumeSource)(nil), ScaleIO: (*v1.ScaleIOVolumeSource)(nil), StorageOS: (*v1.StorageOSVolumeSource)(nil), CSI: (*v1.CSIVolumeSource)(nil), Ephemeral: (*v1.EphemeralVolumeSource)(nil)}}, v1.Volume{Name: "root-volume-0", VolumeSource: v1.VolumeSource{HostPath: (*v1.HostPathVolumeSource)(nil), EmptyDir: (*v1.EmptyDirVolumeSource)(nil), GCEPersistentDisk: (*v1.GCEPersistentDiskVolumeSource)(nil), AWSElasticBlockStore: (*v1.AWSElasticBlockStoreVolumeSource)(nil), GitRepo: (*v1.GitRepoVolumeSource)(nil), Secret: (*v1.SecretVolumeSource)(nil), NFS: (*v1.NFSVolumeSource)(nil), ISCSI: (*v1.ISCSIVolumeSource)(nil), Glusterfs: (*v1.GlusterfsVolumeSource)(nil), PersistentVolumeClaim: (*v1.PersistentVolumeClaimVolumeSource)(nil), RBD: (*v1.RBDVolumeSource)(nil), FlexVolume: (*v1.FlexVolumeSource)(nil), Cinder: (*v1.CinderVolumeSource)(nil), CephFS: (*v1.CephFSVolumeSource)(nil), Flocker: (*v1.FlockerVolumeSource)(nil), DownwardAPI: (*v1.DownwardAPIVolumeSource)(nil), FC: (*v1.FCVolumeSource)(nil), AzureFile: (*v1.AzureFileVolumeSource)(nil), ConfigMap: (*v1.ConfigMapVolumeSource)(0x14000612a40), VsphereVolume: (*v1.VsphereVirtualDiskVolumeSource)(nil), Quobyte: (*v1.QuobyteVolumeSource)(nil), AzureDisk: (*v1.AzureDiskVolumeSource)(nil), PhotonPersistentDisk: (*v1.PhotonPersistentDiskVolumeSource)(nil), Projected: (*v1.ProjectedVolumeSource)(nil), PortworxVolume: (*v1.PortworxVolumeSource)(nil), ScaleIO: (*v1.ScaleIOVolumeSource)(nil), StorageOS: (*v1.StorageOSVolumeSource)(nil), CSI: (*v1.CSIVolumeSource)(nil), Ephemeral: (*v1.EphemeralVolumeSource)(nil)}}}, InitContainers: []v1.Container{v1.Container{Name: "determined-init-container", Image: "", Command: []string{"/run/determined/temp/k8_init_container_entrypoint.sh"}, Args: []string{"2", "/run/determined/temp/tar/src", "/run/determined/temp/tar/dst"}, WorkingDir: "/run/determined/temp/", Ports: []v1.ContainerPort(nil), EnvFrom: []v1.EnvFromSource(nil), Env: []v1.EnvVar(nil), Resources: v1.ResourceRequirements{Limits: v1.ResourceList(nil), Requests: v1.ResourceList(nil)}, VolumeMounts: []v1.VolumeMount{v1.VolumeMount{Name: "archive-volume", ReadOnly: true, MountPath: "/run/determined/temp/tar/src", SubPath: "", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "entrypoint-volume", ReadOnly: true, MountPath: "/run/determined/temp/", SubPath: "", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "additional-files-volume", ReadOnly: false, MountPath: "/run/determined/temp/tar/dst", SubPath: "", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}}, VolumeDevices: []v1.VolumeDevice(nil), LivenessProbe: (*v1.Probe)(nil), ReadinessProbe: (*v1.Probe)(nil), StartupProbe: (*v1.Probe)(nil), Lifecycle: (*v1.Lifecycle)(nil), TerminationMessagePath: "", TerminationMessagePolicy: "", ImagePullPolicy: "IfNotPresent", SecurityContext: (*v1.SecurityContext)(0x14000a06120), Stdin: false, StdinOnce: false, TTY: false}}, Containers: []v1.Container{v1.Container{Name: "determined-fluent-container", Image: "fluent/fluent-bit:1.6", Command: []string{"/fluent-bit/bin/fluent-bit", "-c", "fluent.conf"}, Args: []string(nil), WorkingDir: "/run/determined/fluent/", Ports: []v1.ContainerPort(nil), EnvFrom: []v1.EnvFromSource(nil), Env: []v1.EnvVar(nil), Resources: v1.ResourceRequirements{Limits: v1.ResourceList(nil), Requests: v1.ResourceList(nil)}, VolumeMounts: []v1.VolumeMount{v1.VolumeMount{Name: "det-logs", ReadOnly: false, MountPath: "/run/determined/train/logs", SubPath: "", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "det-fluent", ReadOnly: false, MountPath: "/run/determined/fluent/", SubPath: "", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}}, VolumeDevices: []v1.VolumeDevice(nil), LivenessProbe: (*v1.Probe)(nil), ReadinessProbe: (*v1.Probe)(nil), StartupProbe: (*v1.Probe)(nil), Lifecycle: (*v1.Lifecycle)(nil), TerminationMessagePath: "", TerminationMessagePolicy: "", ImagePullPolicy: "IfNotPresent", SecurityContext: (*v1.SecurityContext)(0x14000a06180), Stdin: false, StdinOnce: false, TTY: false}, v1.Container{Name: "determined-container", Image: "", Command: []string(nil), Args: []string(nil), WorkingDir: "/run/determined/workdir", Ports: []v1.ContainerPort(nil), EnvFrom: []v1.EnvFromSource(nil), Env: []v1.EnvVar{v1.EnvVar{Name: "PYTHONUSERBASE", Value: "/run/determined/pythonuserbase", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_TASK_ID", Value: "", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_RESOURCES_ID", Value: "", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_WORKDIR", Value: "/run/determined/workdir", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_USER", Value: "determined", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_SEGMENT_ENABLED", Value: "false", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_MASTER_ADDR", Value: "0.0.0.0", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_AGENT_ID", Value: "k8agent", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_ALLOCATION_ID", Value: "task", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_SESSION_TOKEN", Value: "", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_CLUSTER_ID", Value: "test", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_MASTER_HOST", Value: "0.0.0.0", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_MASTER_PORT", Value: "32", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_SLOT_IDS", Value: "[0]", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_CONTAINER_ID", Value: "container", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_USER_TOKEN", Value: "bogus", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_USE_TLS", Value: "false", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_MASTER", Value: "http://0.0.0.0:32", ValueFrom: (*v1.EnvVarSource)(nil)}, v1.EnvVar{Name: "DET_K8S_LOG_TO_FILE", Value: "true", ValueFrom: (*v1.EnvVarSource)(nil)}}, Resources: v1.ResourceRequirements{Limits: v1.ResourceList{"nvidia.com/gpu": resource.Quantity{i: resource.int64Amount{value: 1, scale: 0}, d: resource.infDecAmount{Dec: (*inf.Dec)(nil)}, s: "", Format: "DecimalSI"}}, Requests: v1.ResourceList{"nvidia.com/gpu": resource.Quantity{i: resource.int64Amount{value: 1, scale: 0}, d: resource.infDecAmount{Dec: (*inf.Dec)(nil)}, s: "", Format: "DecimalSI"}}}, VolumeMounts: []v1.VolumeMount{v1.VolumeMount{Name: "det-shm-volume", ReadOnly: false, MountPath: "/dev/shm", SubPath: "", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "additional-files-volume", ReadOnly: false, MountPath: "/run/determined", SubPath: "0/run/determined", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "additional-files-volume", ReadOnly: false, MountPath: "/run/determined/info", SubPath: "0/run/determined/info", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "additional-files-volume", ReadOnly: false, MountPath: "/run/determined/pythonuserbase", SubPath: "0/run/determined/pythonuserbase", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "additional-files-volume", ReadOnly: false, MountPath: "/run/determined/workdir", SubPath: "0/run/determined/workdir", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "additional-files-volume", ReadOnly: false, MountPath: "/run/determined/task-logging-setup.sh", SubPath: "1/task-logging-setup.sh", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "additional-files-volume", ReadOnly: false, MountPath: "/run/determined/enrich_task_logs.py", SubPath: "1/enrich_task_logs.py", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "additional-files-volume", ReadOnly: false, MountPath: "/run/determined/task-logging-teardown.sh", SubPath: "1/task-logging-teardown.sh", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "additional-files-volume", ReadOnly: false, MountPath: "/run/determined/task-signal-handling.sh", SubPath: "1/task-signal-handling.sh", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "additional-files-volume", ReadOnly: false, MountPath: "/run/determined/singularity-entrypoint-wrapper.sh", SubPath: "1/singularity-entrypoint-wrapper.sh", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "det-logs", ReadOnly: false, MountPath: "/run/determined/train/logs", SubPath: "", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "det-fluent", ReadOnly: false, MountPath: "/run/determined/fluent/", SubPath: "", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}, v1.VolumeMount{Name: "root-volume-0", ReadOnly: true, MountPath: "/run/determined/etc", SubPath: "", MountPropagation: (*v1.MountPropagationMode)(nil), SubPathExpr: ""}}, VolumeDevices: []v1.VolumeDevice(nil), LivenessProbe: (*v1.Probe)(nil), ReadinessProbe: (*v1.Probe)(nil), StartupProbe: (*v1.Probe)(nil), Lifecycle: (*v1.Lifecycle)(nil), TerminationMessagePath: "", TerminationMessagePolicy: "", ImagePullPolicy: "IfNotPresent", SecurityContext: (*v1.SecurityContext)(0x14000a061e0), Stdin: false, StdinOnce: false, TTY: false}}}}
	podInterface.On("Create", context.TODO(), pod,
		metaV1.CreateOptions{}).Return(&k8sV1.Pod{}, nil).Run(
		func(args mock.Arguments) { pods[newPod.podName] = pod.DeepCopy() })

	ref, _ := system.ActorOf(
		actor.Addr("pod-actor-test"),
		newPod,
	)
	time.Sleep(time.Millisecond * 500)

	return system, newPod, ref, podMap, actorMap, pods
}

var taskContainerFiles = []string{
	"k8_init_container_entrypoint.sh",
	"task-logging-setup.sh",
	"task-logging-teardown.sh",
	"task-signal-handling.sh",
	"enrich_task_logs.py",
	"singularity-entrypoint-wrapper.sh",
}

func setupEntrypoint(t *testing.T) {
	err := etc.SetRootPath(".")
	if err != nil {
		t.Logf("Failed to set root directory")
	}

	for _, file := range taskContainerFiles {
		//nolint:gosec
		f, _ := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		err = f.Close()
		if err != nil {
			t.Logf("failed to close %s", file)
		}
	}
}

func cleanup(t *testing.T) {
	for _, file := range taskContainerFiles {
		err := os.Remove(file)
		if err != nil {
			t.Logf("failed to remove %s", file)
		}
	}
}

func checkReceiveTermination(
	t *testing.T,
	update podStatusUpdate,
	system *actor.System,
	ref *actor.Ref,
	newPod *pod,
	podMap map[string]*mockReceiver,
) {
	system.Ask(ref, update)
	time.Sleep(time.Second)

	assert.Equal(t, podMap["task"].GetLength(), 1)
	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}
	containerMsg, ok := message.(sproto.ResourcesStateChanged)
	if !ok {
		t.Errorf(
			"expected sproto.TaskContainerStateChanged but received %s",
			reflect.TypeOf(message),
		)
	}
	if containerMsg.ResourcesStopped == nil {
		t.Errorf("container started message not present")
	}

	assert.Equal(t, newPod.container.State, cproto.Terminated)
}

func TestResourceCreationFailed(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	const correctMsg = "already exists"

	system, _, ref, podMap, _, _ := createPodWithMockQueue()

	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)
	// Send a second start message to trigger an additional resource creation failure.
	system.Ask(ref, actor.PreStart{})
	time.Sleep(time.Second)

	// We expect two messages in the queue because the pod actor sends itself a stop message.
	assert.Equal(t, podMap["task"].GetLength(), 2)
	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}

	containerMsg, ok := message.(sproto.ContainerLog)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	assert.ErrorContains(t, errors.New(*containerMsg.AuxMessage), correctMsg)
}

func TestReceivePodStatusUpdateTerminated(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	// Pod deleting, but in pending state.
	t.Logf("Testing PodPending status")
	system, newPod, ref, podMap, _, _ := createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	typeMeta := metaV1.TypeMeta{Kind: "rest test"}
	objectMeta := metaV1.ObjectMeta{
		Name:              "test meta",
		DeletionTimestamp: &metaV1.Time{Time: time.Now()},
	}
	pod := k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     k8sV1.PodStatus{Phase: k8sV1.PodPending},
	}

	statusUpdate := podStatusUpdate{updatedPod: &pod}

	checkReceiveTermination(t, statusUpdate, system, ref, newPod, podMap)

	// Pod failed.
	t.Logf("Testing PodFailed status")
	system, newPod, ref, podMap, _, _ = createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)
	pod = k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     k8sV1.PodStatus{Phase: k8sV1.PodFailed},
	}
	statusUpdate = podStatusUpdate{updatedPod: &pod}

	checkReceiveTermination(t, statusUpdate, system, ref, newPod, podMap)

	// Pod succeeded.
	system, newPod, ref, podMap, _, _ = createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)
	pod = k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     k8sV1.PodStatus{Phase: k8sV1.PodSucceeded},
	}
	statusUpdate = podStatusUpdate{updatedPod: &pod}

	checkReceiveTermination(t, statusUpdate, system, ref, newPod, podMap)

	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
}

func TestMultipleContainerTerminate(t *testing.T) {
	// Status update test involving two containers.
	setupEntrypoint(t)
	defer cleanup(t)

	// Pod running with > 1 container, and one terminated.
	t.Logf("Testing two pods with one in terminated state")
	system, newPod, ref, podMap, _, _ := createPodWithMockQueue()
	containerStatuses := []k8sV1.ContainerStatus{
		{
			Name: "test-pod-1",
			State: k8sV1.ContainerState{
				Running: &k8sV1.ContainerStateRunning{},
			},
		},
		{
			Name: "test-pod-2",
			State: k8sV1.ContainerState{
				Terminated: &k8sV1.ContainerStateTerminated{},
			},
		},
	}
	newPod.containerNames = set.FromSlice([]string{"test-pod-1", "test-pod-2"})
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	pod := k8sV1.Pod{
		TypeMeta: metaV1.TypeMeta{Kind: "rest test"},
		ObjectMeta: metaV1.ObjectMeta{
			Name:              "test meta",
			DeletionTimestamp: &metaV1.Time{Time: time.Now()},
		},
		Status: k8sV1.PodStatus{
			Phase:             k8sV1.PodRunning,
			ContainerStatuses: containerStatuses,
		},
	}

	statusUpdate := podStatusUpdate{updatedPod: &pod}
	checkReceiveTermination(t, statusUpdate, system, ref, newPod, podMap)

	// Multiple pods, 1 termination, no deletion timestamp.
	// This results in an error, which causes pod termination and the same outcome.
	t.Logf("Testing two pods with one in terminated state and no deletion timestamp")
	system, newPod, ref, podMap, _, _ = createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	pod = k8sV1.Pod{
		TypeMeta: metaV1.TypeMeta{Kind: "rest test"},
		ObjectMeta: metaV1.ObjectMeta{
			Name: "test meta",
		},
		Status: k8sV1.PodStatus{
			Phase:             k8sV1.PodRunning,
			ContainerStatuses: containerStatuses,
		},
	}

	statusUpdate = podStatusUpdate{updatedPod: &pod}
	checkReceiveTermination(t, statusUpdate, system, ref, newPod, podMap)
}

func TestReceivePodStatusUpdateAssigned(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, newPod, ref, podMap, _, _ := createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	typeMeta := metaV1.TypeMeta{Kind: "rest test"}
	objectMeta := metaV1.ObjectMeta{
		Name: "test meta",
	}
	pod := k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     k8sV1.PodStatus{Phase: k8sV1.PodPending},
	}

	statusUpdate := podStatusUpdate{updatedPod: &pod}

	assert.Equal(t, newPod.container.State, cproto.Assigned)
	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)

	newPod.container.State = cproto.Starting

	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
	assert.Equal(t, newPod.container.State, cproto.Starting)
}

func TestReceivePodStatusUpdateStarting(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, newPod, ref, podMap, _, _ := createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	// Pod status Pending, Pod Scheduled.
	t.Logf("Testing pod scheduled with pending status")
	typeMeta := metaV1.TypeMeta{Kind: "rest test"}
	objectMeta := metaV1.ObjectMeta{
		Name: "test meta",
	}
	condition := k8sV1.PodCondition{
		Type:    k8sV1.PodScheduled,
		Status:  k8sV1.ConditionTrue,
		Message: "This doesn't matter :)",
	}
	status := k8sV1.PodStatus{
		Phase:      k8sV1.PodPending,
		Conditions: []k8sV1.PodCondition{condition},
	}
	pod := k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     status,
	}
	statusUpdate := podStatusUpdate{updatedPod: &pod}

	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)

	assert.Equal(t, podMap["task"].GetLength(), 2)
	assert.Equal(t, newPod.container.State, cproto.Starting)
	podMap["task"].Purge()
	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
	assert.Equal(t, newPod.container.State, cproto.Starting)

	// Pod status Running, but container status Waiting.
	t.Logf("Testing pod running with waiting status")
	system, newPod, ref, podMap, _, _ = createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	containerStatuses := []k8sV1.ContainerStatus{
		{
			Name:  "determined-container",
			State: k8sV1.ContainerState{Waiting: &k8sV1.ContainerStateWaiting{}},
		},
		{
			Name:  "determined-fluent-container",
			State: k8sV1.ContainerState{Waiting: &k8sV1.ContainerStateWaiting{}},
		},
	}
	status = k8sV1.PodStatus{
		Phase:             k8sV1.PodRunning,
		ContainerStatuses: containerStatuses,
	}
	pod = k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     status,
	}
	statusUpdate = podStatusUpdate{updatedPod: &pod}

	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)

	assert.Equal(t, podMap["task"].GetLength(), 2)
	assert.Equal(t, newPod.container.State, cproto.Starting)

	// Pod status running, but no Container State inside.
	t.Logf("Testing pod running with no status")
	system, newPod, ref, podMap, _, _ = createPodWithMockQueue()
	podMap["task"].Purge()
	status = k8sV1.PodStatus{
		Phase: k8sV1.PodRunning,
		ContainerStatuses: []k8sV1.ContainerStatus{
			{Name: "determined-container"},
			{Name: "determined-fluent-container"},
		},
	}
	pod = k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     status,
	}
	statusUpdate = podStatusUpdate{updatedPod: &pod}
	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 2)
	assert.Equal(t, newPod.container.State, cproto.Starting)
}

func TestMultipleContainersRunning(t *testing.T) {
	// Status update test involving two containers.
	setupEntrypoint(t)
	defer cleanup(t)

	// Testing pod with two containers and one doesn't have running state.
	t.Logf("Testing two pods and one doesn't have running state")
	system, newPod, ref, podMap, _, _ := createPodWithMockQueue()
	newPod.container.State = cproto.Starting

	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	typeMeta := metaV1.TypeMeta{Kind: "rest test"}
	objectMeta := metaV1.ObjectMeta{
		Name: "test meta",
	}
	containerStatuses := []k8sV1.ContainerStatus{
		{
			Name:  "determined-container",
			State: k8sV1.ContainerState{Running: &k8sV1.ContainerStateRunning{}},
		},
		{
			Name:  "determined-fluent-container",
			State: k8sV1.ContainerState{Running: &k8sV1.ContainerStateRunning{}},
		},
		{
			Name: "test-pod",
		},
	}
	status := k8sV1.PodStatus{
		Phase:             k8sV1.PodRunning,
		ContainerStatuses: containerStatuses,
	}
	pod := k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     status,
	}
	newPod.containerNames = set.FromSlice([]string{
		"determined-container",
		"determined-fluent-container",
		"test-pod",
	})
	statusUpdate := podStatusUpdate{updatedPod: &pod}

	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
	assert.Equal(t, newPod.container.State, cproto.Starting)

	// Multiple containers, all in running state, results in a running state.
	t.Logf("Testing two pods with running states")
	system, newPod, ref, podMap, _, _ = createPodWithMockQueue()

	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	newPod.container.State = cproto.Starting
	containerStatuses[2] = k8sV1.ContainerStatus{
		Name:  "test-pod-2",
		State: k8sV1.ContainerState{Running: &k8sV1.ContainerStateRunning{}},
	}
	status = k8sV1.PodStatus{
		Phase:             k8sV1.PodRunning,
		ContainerStatuses: containerStatuses,
	}
	pod = k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     status,
	}
	statusUpdate = podStatusUpdate{updatedPod: &pod}
	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)

	assert.Equal(t, podMap["task"].GetLength(), 1)
	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}

	containerMsg, ok := message.(sproto.ResourcesStateChanged)
	if !ok {
		t.Errorf("expected sproto.ResourcesStateChanged but received %s", reflect.TypeOf(message))
	}
	if containerMsg.ResourcesStarted == nil {
		t.Errorf("container started message not present")
	}
}

func TestReceivePodEventUpdate(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, newPod, ref, podMap, _, _ := createPodWithMockQueue()

	object := k8sV1.ObjectReference{Kind: "mock", Namespace: "test", Name: "MockObject"}
	newEvent := k8sV1.Event{
		InvolvedObject: object,
		Reason:         "testing",
		Message:        "0/99 nodes are available: 99 Insufficient cpu",
	}
	newPod.slots = 99
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	system.Ask(ref, podEventUpdate{event: &newEvent})
	time.Sleep(time.Second)

	assert.Equal(t, podMap["task"].GetLength(), 1)
	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}
	correctMsg := fmt.Sprintf("Pod %s: %s", object.Name,
		"Waiting for resources. 0 GPUs are available, 99 GPUs required")

	containerMsg, ok := message.(sproto.ContainerLog)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	assert.Equal(t, *containerMsg.AuxMessage, correctMsg)

	// When container is in Running state, pod actor should not forward message.
	podMap["task"].Purge()
	newPod.container.State = cproto.Running
	system.Ask(ref, podEventUpdate{event: &newEvent})
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)

	// When container is in Terminated state, pod actor should not forward message.
	newPod.container.State = cproto.Terminated
	system.Ask(ref, podEventUpdate{event: &newEvent})
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
}

func TestReceiveContainerLog(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	mockLogMessage := "mock log message"
	system, newPod, ref, podMap, _, _ := createPodWithMockQueue()
	newPod.restore = true
	newPod.container.State = cproto.Running

	podInterface := &mocks.PodInterface{}
	podInterface.On("GetLogs", newPod.podName,
		&k8sV1.PodLogOptions{Container: "determined-container", Follow: true}).Return(
		rest.NewRequestWithClient(&url.URL{}, "", rest.ClientContentConfig{},
			&http.Client{Transport: &mockRoundTripInterface{message: &mockLogMessage}}))

	newPod.podInterface = podInterface
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)
	system.Ask(ref, actor.PreStart{})
	time.Sleep(time.Second)

	assert.Equal(t, podMap["task"].GetLength(), 1)
	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}

	containerMsg, ok := message.(sproto.ContainerLog)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	assert.Equal(t, containerMsg.RunMessage.Value, mockLogMessage)

	// reset state to starting
	newPod.container.State = cproto.Starting
	mockLogMessage = "new mock log message"

	typeMeta := metaV1.TypeMeta{Kind: "running log test"}
	objectMeta := metaV1.ObjectMeta{
		Name: "test meta",
	}
	containerStatuses := []k8sV1.ContainerStatus{
		{
			Name:  "sample-container",
			State: k8sV1.ContainerState{Running: &k8sV1.ContainerStateRunning{}},
		},
	}
	status := k8sV1.PodStatus{
		Phase:             k8sV1.PodRunning,
		ContainerStatuses: containerStatuses,
	}
	pod := k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     status,
	}
	newPod.containerNames = set.FromSlice([]string{
		"sample-container",
	})
	statusUpdate := podStatusUpdate{updatedPod: &pod}

	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 2)
	assert.Equal(t, newPod.container.State, cproto.Running)

	message, err = podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}
	resourceMsg, ok := message.(sproto.ResourcesStateChanged)
	if !ok {
		t.Errorf("expected sproto.ResourcesStateChanged but received %s", reflect.TypeOf(message))
	}
	assert.Equal(t, resourceMsg.Container.State, cproto.Running)

	message, err = podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}
	containerMsg, ok = message.(sproto.ContainerLog)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	assert.Equal(t, containerMsg.RunMessage.Value, mockLogMessage)
}

/*
func TestKillTaskPod(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}
	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)
	system, newPod, ref, _, _ := createPodWithMockQueue(k8sRequestQueue)

	// We take a quick nap immediately so we can purge the start message after it arrives.
	time.Sleep(time.Second)
	assert.Check(t, podInterface.pods[newPod.podName] != nil)
	system.Ask(ref, KillTaskPod{})
	time.Sleep(time.Second)
	assert.Check(t, podInterface.pods[newPod.podName] == nil)
	assert.Check(t, newPod.resourcesDeleted.Load())
}

func TestResourceCreationCancelled(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	podInterface := &mockPodInterface{
		pods:             make(map[string]*k8sV1.Pod),
		operationalDelay: time.Minute * numKubernetesWorkers,
	}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}
	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	for i := 0; i < numKubernetesWorkers; i++ {
		createPodWithMockQueue(k8sRequestQueue)
	}
	time.Sleep(time.Second)
	system, _, ref, podMap, _ := createPodWithMockQueue(k8sRequestQueue)

	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	system.Ask(ref, KillTaskPod{})

	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 1)

	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}

	containerMsg, ok := message.(sproto.ResourcesStateChanged)
	if !ok {
		t.Errorf("expected sproto.TaskContainerStateChanged but received %s",
			reflect.TypeOf(message))
	}

	var correctContainerStarted *sproto.ResourcesStarted = nil
	correctFailType := "task failed without an associated exit code"
	correctErrMsg := "pod actor exited while pod was running"
	var correctCode *sproto.ExitCode = nil

	assert.Equal(t, containerMsg.ResourcesStarted, correctContainerStarted)
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.FailureType,
		sproto.FailureType(correctFailType))
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.ErrMsg, correctErrMsg)
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.ExitCode, correctCode)
}
*/

func TestResourceDeletionFailed(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, newPod, ref, podMap, _, pods := createPodWithMockQueue()

	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)
	delete(pods, newPod.podName)
	system.Ask(ref, KillTaskPod{})
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 1)

	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}

	containerMsg, ok := message.(sproto.ResourcesStateChanged)
	if !ok {
		t.Errorf("expected sproto.TaskContainerStateChanged but received %s",
			reflect.TypeOf(message))
	}

	var correctContainerStarted *sproto.ResourcesStarted = nil
	var correctCode *sproto.ExitCode = nil

	assert.Equal(t, containerMsg.ResourcesStarted, correctContainerStarted)
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.FailureType,
		sproto.FailureType("task failed without an associated exit code"))
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.ErrMsg,
		"pod actor exited while pod was running")
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.ExitCode, correctCode)
}

func TestGetPodNodeInfo(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, newPod, ref, podMap, _, _ := createPodWithMockQueue()
	newPod.slots = 99
	time.Sleep(time.Second)

	podMap["task"].Purge()
	podMap["cluster"].Purge()
	podMap["resource"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)
	assert.Equal(t, podMap["cluster"].GetLength(), 0)
	assert.Equal(t, podMap["resource"].GetLength(), 0)

	data := system.Ask(ref, getPodNodeInfo{})
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
	assert.Equal(t, podMap["cluster"].GetLength(), 0)
	assert.Equal(t, podMap["resource"].GetLength(), 0)

	podInfo, ok := data.Get().(podNodeInfo)
	if !ok {
		t.Errorf("expected podNodeInfo but received %s", reflect.TypeOf(data))
	}

	assert.Equal(t, podInfo.nodeName, newPod.pod.Spec.NodeName)
	assert.Equal(t, podInfo.numSlots, newPod.slots)
}
