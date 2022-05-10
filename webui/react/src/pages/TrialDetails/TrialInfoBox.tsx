import React, { useMemo } from 'react';

import CheckpointModalTrigger from 'components/CheckpointModalTrigger';
import Grid, { GridMode } from 'components/Grid';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import TimeAgo from 'components/TimeAgo';
import { humanReadableBytes } from 'shared/utils/string';
import { ShirtSize } from 'themes';
import {
  CheckpointState, CheckpointWorkload, CheckpointWorkloadExtended, ExperimentBase, TrialDetails,
} from 'types';
import { checkpointSize } from 'utils/workload';

interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

const TrialInfoBox: React.FC<Props> = ({ trial, experiment }: Props) => {

  const bestCheckpoint: CheckpointWorkloadExtended | undefined = useMemo(() => {
    if (!trial) return;
    const cp = trial.bestAvailableCheckpoint;
    if (!cp) return;

    return {
      ...cp,
      experimentId: trial.experimentId,
      trialId: trial.id,
    };
  }, [ trial ]);

  const totalCheckpointsSize = useMemo(() => {
    const totalBytes = trial?.workloads
      .filter(step => step.checkpoint
        && step.checkpoint.state === CheckpointState.Completed)
      .map(step => checkpointSize(step.checkpoint as CheckpointWorkload))
      .reduce((acc, cur) => acc + cur, 0);
    if (!totalBytes) return;
    return humanReadableBytes(totalBytes);
  }, [ trial?.workloads ]);

  return (
    <Section>
      <Grid gap={ShirtSize.medium} minItemWidth={180} mode={GridMode.AutoFill}>
        {trial?.runnerState && (
          <OverviewStats title="Last Runner State">
            {trial.runnerState}
          </OverviewStats>
        )}
        {trial?.startTime && (
          <OverviewStats title="Start Time">
            <TimeAgo datetime={trial.startTime} />
          </OverviewStats>
        )}
        {totalCheckpointsSize && (
          <OverviewStats title="Total Checkpoint Size">
            {totalCheckpointsSize}
          </OverviewStats>
        )}
        {bestCheckpoint && (
          <CheckpointModalTrigger
            checkpoint={bestCheckpoint}
            experiment={experiment}
            title="Best Checkpoint">
            <OverviewStats clickable title="Best Checkpoint">
              Batch {bestCheckpoint.totalBatches}
            </OverviewStats>
          </CheckpointModalTrigger>
        )}
      </Grid>
    </Section>
  );
};

export default TrialInfoBox;
