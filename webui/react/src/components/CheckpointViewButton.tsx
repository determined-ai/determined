import React, { PropsWithChildren, useCallback } from 'react';

import useModalCheckpoint from 'hooks/useModal/useModalCheckpoint';
import {
  CheckpointDetail, CheckpointWorkloadExtended, ExperimentBase,
} from 'types';

interface Props {
  checkpoint: CheckpointWorkloadExtended | CheckpointDetail;
  experiment: ExperimentBase;
  title: string;
}

const CheckpointViewButton: React.FC<Props> = (
  {
    checkpoint,
    experiment,
    title,
    ...props
  }: PropsWithChildren<Props>,
) => {
  const { modalOpen: openModalDelete } =
  useModalCheckpoint({ checkpoint: checkpoint, config: experiment.config, title: title });
  const handleModalCheckpointClick = useCallback(() => openModalDelete(), [ openModalDelete ]);

  return (
    <span onClick={() => handleModalCheckpointClick()}>
      {props.children}
    </span>
  );
};

export default CheckpointViewButton;
