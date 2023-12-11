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
      <registerModal.Component onClose={() => 1} />
      <modelCreateModal.Component onClose={() => 1} />
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
