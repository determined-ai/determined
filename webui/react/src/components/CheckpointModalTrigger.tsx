import Button from 'hew/Button';
import Icon from 'hew/Icon';
import { Loadable } from 'hew/utils/loadable';
import React, { useCallback } from 'react';

import { useCheckpointFlow } from 'hooks/useCheckpointFlow';
import {
  CheckpointWorkloadExtended,
  CoreApiGenericCheckpoint,
  ExperimentBase,
  ModelItem,
} from 'types';

import css from './CheckpointModalTrigger.module.scss';

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
  const {
    checkpointModalComponent,
    modelCreateModalComponent,
    registerModalComponent,
    openCheckpoint,
  } = useCheckpointFlow({
    checkpoint: checkpoint,
    config: experiment.config,
    models,
    title: title,
  });

  const handleModalCheckpointClick = useCallback(() => {
    openCheckpoint();
  }, [openCheckpoint]);

  return (
    <>
      <span className={css.base} onClick={handleModalCheckpointClick}>
        {children !== undefined ? (
          children
        ) : (
          <Button
            aria-label="View Checkpoint"
            icon={<Icon name="checkpoint" showTooltip title="View Checkpoint" />}
          />
        )}
      </span>
      {checkpointModalComponent}
      {modelCreateModalComponent}
      {registerModalComponent}
    </>
  );
};

export default CheckpointModalTrigger;
