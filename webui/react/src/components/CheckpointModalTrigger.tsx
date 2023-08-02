import React, { useCallback } from 'react';

import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import ModelCreateModal from 'components/ModelCreateModal';
import useModalCheckpointRegister from 'hooks/useModal/Checkpoint/useModalCheckpointRegister';
import { ModalCloseReason } from 'hooks/useModal/useModal';
import { CheckpointWorkloadExtended, CoreApiGenericCheckpoint, ExperimentBase } from 'types';

import CheckpointModalComponent from './CheckpointModal';

interface Props {
  checkpoint: CheckpointWorkloadExtended | CoreApiGenericCheckpoint;
  children?: React.ReactNode;
  experiment: ExperimentBase;
  title: string;
}

const CheckpointModalTrigger: React.FC<Props> = ({
  checkpoint,
  experiment,
  title,
  children,
}: Props) => {
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
      if (reason === ModalCloseReason.Ok && checkpoint.uuid) {
        openModalCheckpointRegister({ checkpoints: checkpoint.uuid });
      }
    },
    [checkpoint, openModalCheckpointRegister],
  );

  const handleModalCheckpointClick = useCallback(() => {
    checkpointModal.open();
  }, [checkpointModal]);

  return (
    <>
      <span onClick={handleModalCheckpointClick}>
        {children !== undefined ? (
          children
        ) : (
          <Button
            aria-label="View Checkpoint"
            icon={<Icon name="checkpoint" showTooltip title="View Checkpoint" />}
          />
        )}
      </span>
      {modalCheckpointRegisterContextHolder}
      <modelCreateModal.Component onClose={handleOnCloseCreateModel} />
      <checkpointModal.Component
        checkpoint={checkpoint}
        config={experiment.config}
        title={title}
        onClose={handleOnCloseCheckpoint}
      />
    </>
  );
};

export default CheckpointModalTrigger;
