import Button from 'hew/Button';
import Icon from 'hew/Icon';
import { useModal } from 'hew/Modal';
import React, { useCallback } from 'react';

import ModelCreateModal from 'components/ModelCreateModal';
import RegisterCheckpointModal from 'components/RegisterCheckpointModal';
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
      if (reason === 'Ok' && checkpoint.uuid) {
        registerModal.open();
        console.log({ checkpoints: checkpoint.uuid });
      }
    },
    [checkpoint, registerModal],
  );

  const handleModalCheckpointClick = useCallback(() => {
    checkpointModal.open();
  }, [checkpointModal]);

  const handleOnCloseRegister = useCallback((_reason?: string, checkpoints?: string[]) => {
    if (checkpoints) modelCreateModal.open();
  }, []);

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
      <registerModal.Component onClose={handleOnCloseRegister} />
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
