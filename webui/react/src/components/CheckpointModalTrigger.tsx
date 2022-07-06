import { Button, Tooltip } from 'antd';
import React, { PropsWithChildren, useCallback } from 'react';

import useModalCheckpoint from 'hooks/useModal/Checkpoint/useModalCheckpoint';
import useModalCheckpointRegister from 'hooks/useModal/Checkpoint/useModalCheckpointRegister';
import useModalModelCreate from 'hooks/useModal/Model/useModalModelCreate';
import { ModalCloseReason } from 'hooks/useModal/useModal';
import Icon from 'shared/components/Icon/Icon';
import { CheckpointWorkloadExtended, ExperimentBase } from 'types';

interface Props {
  checkpoint: CheckpointWorkloadExtended;
  chidren?: JSX.Element
  experiment: ExperimentBase;
  title: string;
}

const CheckpointModalTrigger: React.FC<Props> = ({
  checkpoint,
  experiment,
  title,
  children,
}: PropsWithChildren<Props>) => {
  const {
    contextHolder: modalModelCreateContextHolder,
    modalOpen: openModalCreateModel,
  } = useModalModelCreate();

  const handleOnCloseCheckpointRegister = useCallback((
    reason?: ModalCloseReason,
    checkpointUuid?: string,
  ) => {
    if (checkpointUuid) openModalCreateModel({ checkpointUuid });
  }, [ openModalCreateModel ]);

  const {
    contextHolder: modalCheckpointRegisterContextHolder,
    modalOpen: openModalCheckpointRegister,
  } = useModalCheckpointRegister({ onClose: handleOnCloseCheckpointRegister });

  const handleOnCloseCheckpoint = useCallback((reason?: ModalCloseReason) => {
    if (reason === ModalCloseReason.Ok && checkpoint.uuid) {
      openModalCheckpointRegister({ checkpointUuid: checkpoint.uuid });
    }
  }, [ checkpoint, openModalCheckpointRegister ]);

  const {
    contextHolder: modalCheckpointContextHolder,
    modalOpen: openModalCheckpoint,
  } = useModalCheckpoint({
    checkpoint,
    config: experiment.config,
    onClose: handleOnCloseCheckpoint,
    title,
  });

  const handleModalCheckpointClick = useCallback(() => {
    openModalCheckpoint();
  }, [ openModalCheckpoint ]);

  return (
    <>
      <span onClick={handleModalCheckpointClick}>
        {children !== undefined ? children : (
          <Tooltip title="View Checkpoint">
            <Button
              aria-label="View Checkpoint"
              icon={<Icon name="checkpoint" />}
            />
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
