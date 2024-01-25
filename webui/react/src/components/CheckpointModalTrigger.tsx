import Button from 'hew/Button';
import Icon from 'hew/Icon';
import { ModalCloseReason, useModal } from 'hew/Modal';
import { Loadable } from 'hew/utils/loadable';
import React, { useCallback, useState } from 'react';

import ModelCreateModal from 'components/ModelCreateModal';
import RegisterCheckpointModal from 'components/RegisterCheckpointModal';
import {
  CheckpointWorkloadExtended,
  CoreApiGenericCheckpoint,
  ExperimentBase,
  ModelItem,
} from 'types';

import CheckpointModalComponent from './CheckpointModal';

interface Props {
  checkpoint: CheckpointWorkloadExtended | CoreApiGenericCheckpoint;
  children?: React.ReactNode;
  experiment: ExperimentBase;
  title: string;
  models: Loadable<ModelItem[]>;
}

const CheckpointModalTrigger: React.FC<Props> = ({
  checkpoint,
  experiment,
  title,
  children,
  models,
}: Props) => {
  const modelCreateModal = useModal(ModelCreateModal);
  const checkpointModal = useModal(CheckpointModalComponent);

  const registerModal = useModal(RegisterCheckpointModal);

  const [selectedModelName, setSelectedModelName] = useState<string>();

  const handleOnCloseCreateModel = useCallback(
    (modelName?: string) => {
      if (modelName) {
        setSelectedModelName(modelName);
        registerModal.open();
      }
    },
    [setSelectedModelName, registerModal],
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
      <registerModal.Component
        checkpoints={checkpoint.uuid ? [checkpoint.uuid] : []}
        closeModal={registerModal.close}
        modelName={selectedModelName}
        models={models}
        openModelModal={modelCreateModal.open}
      />
      <checkpointModal.Component
        checkpoint={checkpoint}
        config={experiment.config}
        title={title}
        onClose={(reason?: ModalCloseReason) => {
          if (reason === 'Ok') registerModal.open();
        }}
      />
      <modelCreateModal.Component onClose={handleOnCloseCreateModel} />
    </>
  );
};

export default CheckpointModalTrigger;
