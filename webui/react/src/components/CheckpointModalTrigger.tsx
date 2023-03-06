import React, { useCallback } from 'react';

import Button from 'components/kit/Button';
import Tooltip from 'components/kit/Tooltip';
import useModalCheckpoint from 'hooks/useModal/Checkpoint/useModalCheckpoint';
import useModalCheckpointRegister from 'hooks/useModal/Checkpoint/useModalCheckpointRegister';
import useModalModelCreate from 'hooks/useModal/Model/useModalModelCreate';
import Icon from 'shared/components/Icon/Icon';
import { ModalCloseReason } from 'shared/hooks/useModal/useModal';
import { CheckpointWorkloadExtended, CoreApiGenericCheckpoint, ExperimentBase } from 'types';

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
  const {
    contextHolder: modalCheckpointRegisterContextHolder,
    modalOpen: openModalCheckpointRegister,
  } = useModalCheckpointRegister({
    onClose: (reason?: ModalCloseReason, checkpoints?: string[]) => {
      if (checkpoints) openModalCreateModel({ checkpoints });
    },
  });

  const handleOnCloseCreateModel = useCallback(
    (reason?: ModalCloseReason, checkpoints?: string[], modelName?: string) => {
      if (checkpoints) openModalCheckpointRegister({ checkpoints, selectedModelName: modelName });
    },
    [openModalCheckpointRegister],
  );

  const { contextHolder: modalModelCreateContextHolder, modalOpen: openModalCreateModel } =
    useModalModelCreate({ onClose: handleOnCloseCreateModel });

  const handleOnCloseCheckpoint = useCallback(
    (reason?: ModalCloseReason) => {
      if (reason === ModalCloseReason.Ok && checkpoint.uuid) {
        openModalCheckpointRegister({ checkpoints: checkpoint.uuid });
      }
    },
    [checkpoint, openModalCheckpointRegister],
  );

  const { contextHolder: modalCheckpointContextHolder, modalOpen: openModalCheckpoint } =
    useModalCheckpoint({
      checkpoint,
      config: experiment.config,
      onClose: handleOnCloseCheckpoint,
      title,
    });

  const handleModalCheckpointClick = useCallback(() => {
    openModalCheckpoint();
  }, [openModalCheckpoint]);

  return (
    <>
      <span onClick={handleModalCheckpointClick}>
        {children !== undefined ? (
          children
        ) : (
          <Tooltip title="View Checkpoint">
            <Button aria-label="View Checkpoint" icon={<Icon name="checkpoint" />} />
          </Tooltip>
        )}
      </span>
      {modalCheckpointContextHolder}
      {modalCheckpointRegisterContextHolder}
      {modalModelCreateContextHolder}
    </>
  );
};

export default CheckpointModalTrigger;
