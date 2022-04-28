import { Button, Tooltip } from 'antd';
import React, { useCallback } from 'react';
import Icon from 'components/Icon';
import useModalCheckpoint from 'hooks/useModal/useModalCheckpoint';
import {
  CheckpointDetail, ExperimentBase
}  from 'types';

interface Props {
  checkpoint: CheckpointDetail;
  experiment: ExperimentBase;
  title: string;
}

const CheckpointViewButton: React.FC<Props> = (
  {
    checkpoint,
    experiment,
    title,
  }: Props,
) => {
  const { modalOpen: openModalDelete } =
  useModalCheckpoint({ checkpoint: checkpoint, config: experiment.config, title: title });
  const handleModalCheckpointClick = useCallback(() => openModalDelete(), [ openModalDelete ]);

  return (
    <Tooltip title="View Checkpoint">
      <Button
        aria-label="View Checkpoint"
        icon={<Icon name="checkpoint" />}
        onClick={() => handleModalCheckpointClick()}
      />
    </Tooltip>
  );
};

export default CheckpointViewButton;