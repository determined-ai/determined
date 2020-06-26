package kubernetes

import (
	"fmt"
	"path"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/container"

	"github.com/pkg/errors"

	"github.com/docker/docker/api/types/mount"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func configureMountPropagation(b *mount.BindOptions) *v1.MountPropagationMode {
	if b != nil {
		switch b.Propagation {
		case mount.PropagationPrivate:
			p := v1.MountPropagationNone
			return &p
		case mount.PropagationRSlave:
			p := v1.MountPropagationHostToContainer
			return &p
		case mount.PropagationRShared:
			p := v1.MountPropagationBidirectional
			return &p
		default:
			return nil
		}
	}

	return nil
}

func dockerMountsToHostVolumes(dockerMounts []mount.Mount) ([]v1.VolumeMount, []v1.Volume) {
	volumeMounts := make([]v1.VolumeMount, 0, len(dockerMounts))
	volumes := make([]v1.Volume, 0, len(dockerMounts))

	for idx, d := range dockerMounts {
		name := fmt.Sprintf("det-host-volume-%d", idx)
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:             name,
			ReadOnly:         d.ReadOnly,
			MountPath:        d.Target,
			MountPropagation: configureMountPropagation(d.BindOptions),
		})
		volumes = append(volumes, v1.Volume{
			Name: name,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: d.Source,
				},
			},
		})
	}

	return volumeMounts, volumes
}

func configureShmVolume(_ int64) (v1.VolumeMount, v1.Volume) {
	// Kubernetes does not support a native way to set shm size for
	// containers. The workaround for this is to create an emptyDir
	// volume and mount it to /dev/shm.
	volumeName := "det-shm-volume"
	volumeMount := v1.VolumeMount{
		Name:      volumeName,
		ReadOnly:  false,
		MountPath: "/det/shm",
	}
	volume := v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{
			Medium: v1.StorageMediumMemory,
		}},
	}
	return volumeMount, volume
}

func createConfigMapSpec(
	namePrefix string,
	data map[string][]byte,
	namespace string,
) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: namePrefix,
			Namespace:    namespace,
		},
		BinaryData: data,
	}
}

func startConfigMap(
	ctx *actor.Context,
	configMapSpec *v1.ConfigMap,
	configMapInterface typedV1.ConfigMapInterface,
) (*v1.ConfigMap, error) {
	configMap, err := configMapInterface.Create(configMapSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create configMap")
	}
	ctx.Log().Infof("create configMap %s", configMap.Name)

	return configMap, nil
}

func configureAdditionalFilesVolumes(
	archiveConfigMap *v1.ConfigMap,
	entryPointConfigMap *v1.ConfigMap,
	runArchives []container.RunArchive,
) ([]v1.VolumeMount, []v1.VolumeMount, []v1.Volume) {
	initContainerVolumeMounts := make([]v1.VolumeMount, 0, 3)
	mainContainerVolumeMounts := make([]v1.VolumeMount, 0)
	volumes := make([]v1.Volume, 0, 3)

	// In order to inject additional files into k8 pods, we un-tar the archives
	// in an initContainer from a configMap to an emptyDir, and then mount the
	// emptyDir into the main container.

	archiveVolumeName := "archive-volume"
	archiveVolume := v1.Volume{
		Name: archiveVolumeName,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{Name: archiveConfigMap.Name},
			},
		},
	}
	volumes = append(volumes, archiveVolume)
	archiveVolumeMount := v1.VolumeMount{
		Name:      archiveVolumeName,
		MountPath: initContainerTarSrcPath,
		ReadOnly:  true,
	}
	initContainerVolumeMounts = append(initContainerVolumeMounts, archiveVolumeMount)

	entryPointVolumeName := "entrypoint-volume"
	var entryPointVolumeMode int32 = 0700
	entryPointVolume := v1.Volume{
		Name: entryPointVolumeName,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{Name: entryPointConfigMap.Name},
				DefaultMode:          &entryPointVolumeMode,
			},
		},
	}
	volumes = append(volumes, entryPointVolume)
	entrypointVolumeMount := v1.VolumeMount{
		Name:      entryPointVolumeName,
		MountPath: initContainerWorkDir,
		ReadOnly:  true,
	}
	initContainerVolumeMounts = append(initContainerVolumeMounts, entrypointVolumeMount)

	additionalFilesVolumeName := "additional-files-volume"
	dstVolume := v1.Volume{
		Name:         additionalFilesVolumeName,
		VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
	}
	volumes = append(volumes, dstVolume)
	dstVolumeMount := v1.VolumeMount{
		Name:      additionalFilesVolumeName,
		MountPath: initContainerTarDstPath,
		ReadOnly:  false,
	}
	initContainerVolumeMounts = append(initContainerVolumeMounts, dstVolumeMount)

	for idx, runArchive := range runArchives {
		for _, item := range runArchive.Archive {
			mainContainerVolumeMounts = append(mainContainerVolumeMounts, v1.VolumeMount{
				Name:      additionalFilesVolumeName,
				MountPath: path.Join(runArchive.Path, item.Path),
				SubPath:   path.Join(fmt.Sprintf("%d", idx), item.Path),
			})
		}
	}

	return initContainerVolumeMounts, mainContainerVolumeMounts, volumes
}
