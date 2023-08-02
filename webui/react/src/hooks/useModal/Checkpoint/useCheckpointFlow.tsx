import { ReactElement, useCallback, useMemo } from 'react';

import CheckpointModalComponent from 'components/CheckpointModal';
import { useModal } from 'components/kit/Modal';
import ModelCreateModal from 'components/ModelCreateModal';
import { ModalCloseReason } from 'hooks/useModal/useModal';
import { CheckpointWorkloadExtended, CoreApiGenericCheckpoint, ExperimentConfig } from 'types';

import useModalCheckpointRegister from './useModalCheckpointRegister';

interface Return {
  checkpointModalComponent: React.ReactNode;
  contextHolders: ReactElement[];
  openCheckpoint: () => void;
  modelCreateModalComponent: React.ReactNode;
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
  const {
    contextHolder: modalCheckpointRegisterContextHolder,
    modalOpen: openModalCheckpointRegister,
  } = useModalCheckpointRegister({
    onClose: (reason?: ModalCloseReason, checkpoints?: string[]) => {
      // TODO: fix the behavior along with checkpoint modal migration
      // It used to open checkpoint modal again after creating a model,
      // but it doesn't with new create model modal since we don't use context holder anymore.
      // This should be able to fix it along with checkpoint modal migration.
      if (checkpoints) modelCreateModal.open();
    },
  });

  const handleOnCloseCreateModel = useCallback(
    (reason?: ModalCloseReason, checkpoints?: string[], modelName?: string) => {
      if (checkpoints) openModalCheckpointRegister({ checkpoints, selectedModelName: modelName });
    },
    [openModalCheckpointRegister],
  );

  const handleOnCloseCheckpoint = useCallback(
    (reason?: ModalCloseReason) => {
      if (reason === ModalCloseReason.Ok && checkpoint?.uuid) {
        openModalCheckpointRegister({ checkpoints: checkpoint.uuid });
      }
    },
    [checkpoint, openModalCheckpointRegister],
  );

  const openCheckpoint = useCallback(() => {
    checkpointModal.open();
  }, [checkpointModal]);

  const contextHolders = useMemo(
    () => [modalCheckpointRegisterContextHolder],
    [modalCheckpointRegisterContextHolder],
  );

  return {
    checkpointModalComponent: (
      <checkpointModal.Component
        checkpoint={checkpoint}
        config={config}
        title={title}
        onClose={handleOnCloseCheckpoint}
      />
    ),
    contextHolders,
    modelCreateModalComponent: <modelCreateModal.Component onClose={handleOnCloseCreateModel} />,
    openCheckpoint,
  };
};
