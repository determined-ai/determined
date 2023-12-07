import { useModal } from 'hew/Modal';
import { useCallback } from 'react';

import CheckpointModalComponent from 'components/CheckpointModal';
import ModelCreateModal from 'components/ModelCreateModal';
import RegisterCheckpointModal from 'components/RegisterCheckpointModal';
import { CheckpointWorkloadExtended, CoreApiGenericCheckpoint, ExperimentConfig } from 'types';

interface Return {
  checkpointModalComponent: React.ReactNode;
  openCheckpoint: () => void;
  modelCreateModalComponent: React.ReactNode;
  registerModalComponent: React.ReactNode;
}

export const useCheckpointFlow = ({
  checkpoint,
  config,
  title,
}: {
  checkpoint?: CheckpointWorkloadExtended | CoreApiGenericCheckpoint;
  config: ExperimentConfig;
  title: string;
}): Return => {
  const modelCreateModal = useModal(ModelCreateModal);
  const checkpointModal = useModal(CheckpointModalComponent);
  const registerModal = useModal(RegisterCheckpointModal);

  const handleOnCloseCreateModel = useCallback(
    (_reason?: string, checkpoints?: string[], modelName?: string) => {
      if (checkpoints) registerModal.open();
      console.log({ checkpoints, selectedModelName: modelName });
    },
    [registerModal],
  );

  const handleOnCloseCheckpoint = useCallback(
    (reason?: string) => {
      if (reason === 'Ok' && checkpoint?.uuid) {
        registerModal.open();
        console.log({ checkpoints: checkpoint.uuid });
      }
    },
    [checkpoint, registerModal],
  );

  const openCheckpoint = useCallback(() => {
    checkpointModal.open();
  }, [checkpointModal]);

  return {
    checkpointModalComponent: (
      <checkpointModal.Component
        checkpoint={checkpoint}
        config={config}
        title={title}
        onClose={handleOnCloseCheckpoint}
      />
    ),
    modelCreateModalComponent: <modelCreateModal.Component onClose={handleOnCloseCreateModel} />,
    registerModalComponent: (
      <registerModal.Component onClose={handleOnCloseCreateModel}/>
    ),
    openCheckpoint,
  };
};
