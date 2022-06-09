import React, { useMemo, useState } from 'react';

import CheckpointModalTrigger from 'components/CheckpointModalTrigger';
import Grid, { GridMode } from 'components/Grid';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import TimeAgo from 'components/TimeAgo';
import { getTrialWorkloads } from 'services/api';
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

  const [ totalCheckpointsSize, setTotalCheckpointsSize ] = useState<number>(0);
  useMemo(async () => {
    if (!trial) {
      return;
    }
    const data = await getTrialWorkloads({
      filter: 'Has Checkpoint',
      id: trial.id,
      limit: 1000,
    });
    const checkpointWorkloads = data.workloads;
    const totalBytes = checkpointWorkloads
      .filter(step => step.checkpoint
        && step.checkpoint.state === CheckpointState.Completed)
      .map(step => checkpointSize(step.checkpoint as CheckpointWorkload))
      .reduce((acc, cur) => acc + cur, 0);
    if (!totalBytes) return;
    setTotalCheckpointsSize(totalBytes);
  }, [ trial, trial?.id ]);

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
            {humanReadableBytes(totalCheckpointsSize)}
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
