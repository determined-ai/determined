package kubernetes

import (
	"fmt"
	"path"

	"github.com/determined-ai/determined/master/pkg/etc"

	"github.com/pkg/errors"

	"github.com/docker/docker/api/types/mount"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/container"
)

func configureMountPropagation(b *mount.BindOptions) *k8sV1.MountPropagationMode {
	if b == nil {
		return nil
	}

	switch b.Propagation {
	case mount.PropagationPrivate:
		p := k8sV1.MountPropagationNone
		return &p
	case mount.PropagationRSlave:
		p := k8sV1.MountPropagationHostToContainer
		return &p
	case mount.PropagationRShared:
		p := k8sV1.MountPropagationBidirectional
		return &p
	default:
		return nil
	}
}

func dockerMountsToHostVolumes(dockerMounts []mount.Mount) ([]k8sV1.VolumeMount, []k8sV1.Volume) {
	volumeMounts := make([]k8sV1.VolumeMount, 0, len(dockerMounts))
	volumes := make([]k8sV1.Volume, 0, len(dockerMounts))

	for idx, d := range dockerMounts {
		name := fmt.Sprintf("det-host-volume-%d", idx)
		volumeMounts = append(volumeMounts, k8sV1.VolumeMount{
			Name:             name,
			ReadOnly:         d.ReadOnly,
			MountPath:        d.Target,
			MountPropagation: configureMountPropagation(d.BindOptions),
		})
		volumes = append(volumes, k8sV1.Volume{
			Name: name,
			VolumeSource: k8sV1.VolumeSource{
				HostPath: &k8sV1.HostPathVolumeSource{
					Path: d.Source,
				},
			},
		})
	}

	return volumeMounts, volumes
}

func configureShmVolume(_ int64) (k8sV1.VolumeMount, k8sV1.Volume) {
	// Kubernetes does not support a native way to set shm size for
	// containers. The workaround for this is to create an emptyDir
	// volume and mount it to /dev/shm.
	volumeName := "det-shm-volume"
	volumeMount := k8sV1.VolumeMount{
		Name:      volumeName,
		ReadOnly:  false,
		MountPath: "/dev/shm",
	}
	volume := k8sV1.Volume{
		Name: volumeName,
		VolumeSource: k8sV1.VolumeSource{EmptyDir: &k8sV1.EmptyDirVolumeSource{
			Medium: k8sV1.StorageMediumMemory,
		}},
	}
	return volumeMount, volume
}

func createConfigMapSpec(
	namePrefix string,
	data map[string][]byte,
	namespace string,
	taskID string,
) *k8sV1.ConfigMap {
	return &k8sV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			GenerateName: namePrefix,
			Namespace:    namespace,
			Labels:       map[string]string{determinedLabel: taskID},
		},
		BinaryData: data,
	}
}

func startConfigMap(
	ctx *actor.Context,
	configMapSpec *k8sV1.ConfigMap,
	configMapInterface typedV1.ConfigMapInterface,
) (*k8sV1.ConfigMap, error) {
	configMap, err := configMapInterface.Create(configMapSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create configMap")
	}
	ctx.Log().Infof("created configMap %s", configMap.Name)

	return configMap, nil
}

func configureAdditionalFilesVolumes(
	configMap *k8sV1.ConfigMap,
	runArchives []container.RunArchive,
) ([]k8sV1.VolumeMount, []k8sV1.VolumeMount, []k8sV1.Volume) {
	initContainerVolumeMounts := make([]k8sV1.VolumeMount, 0)
	mainContainerVolumeMounts := make([]k8sV1.VolumeMount, 0)
	volumes := make([]k8sV1.Volume, 0)

	// In order to inject additional files into k8 pods, we un-tar the archives
	// in an initContainer from a configMap to an emptyDir, and then mount the
	// emptyDir into the main container.

	archiveVolumeName := "archive-volume"
	archiveVolume := k8sV1.Volume{
		Name: archiveVolumeName,
		VolumeSource: k8sV1.VolumeSource{
			ConfigMap: &k8sV1.ConfigMapVolumeSource{
				LocalObjectReference: k8sV1.LocalObjectReference{Name: configMap.Name},
			},
		},
	}
	volumes = append(volumes, archiveVolume)
	archiveVolumeMount := k8sV1.VolumeMount{
		Name:      archiveVolumeName,
		MountPath: initContainerTarSrcPath,
		ReadOnly:  true,
	}
	initContainerVolumeMounts = append(initContainerVolumeMounts, archiveVolumeMount)

	entryPointVolumeName := "entrypoint-volume"
	var entryPointVolumeMode int32 = 0700
	entryPointVolume := k8sV1.Volume{
		Name: entryPointVolumeName,
		VolumeSource: k8sV1.VolumeSource{
			ConfigMap: &k8sV1.ConfigMapVolumeSource{
				LocalObjectReference: k8sV1.LocalObjectReference{Name: configMap.Name},
				Items: []k8sV1.KeyToPath{{
					Key:  etc.K8InitContainerEntryScriptResource,
					Path: etc.K8InitContainerEntryScriptResource,
				}},
				DefaultMode: &entryPointVolumeMode,
			},
		},
	}
	volumes = append(volumes, entryPointVolume)
	entrypointVolumeMount := k8sV1.VolumeMount{
		Name:      entryPointVolumeName,
		MountPath: initContainerWorkDir,
		ReadOnly:  true,
	}
	initContainerVolumeMounts = append(initContainerVolumeMounts, entrypointVolumeMount)

	additionalFilesVolumeName := "additional-files-volume"
	dstVolume := k8sV1.Volume{
		Name:         additionalFilesVolumeName,
		VolumeSource: k8sV1.VolumeSource{EmptyDir: &k8sV1.EmptyDirVolumeSource{}},
	}
	volumes = append(volumes, dstVolume)
	dstVolumeMount := k8sV1.VolumeMount{
		Name:      additionalFilesVolumeName,
		MountPath: initContainerTarDstPath,
		ReadOnly:  false,
	}
	initContainerVolumeMounts = append(initContainerVolumeMounts, dstVolumeMount)

	for idx, runArchive := range runArchives {
		for _, item := range runArchive.Archive {
			mainContainerVolumeMounts = append(mainContainerVolumeMounts, k8sV1.VolumeMount{
				Name:      additionalFilesVolumeName,
				MountPath: path.Join(runArchive.Path, item.Path),
				SubPath:   path.Join(fmt.Sprintf("%d", idx), item.Path),
			})
		}
	}

	return initContainerVolumeMounts, mainContainerVolumeMounts, volumes
}
