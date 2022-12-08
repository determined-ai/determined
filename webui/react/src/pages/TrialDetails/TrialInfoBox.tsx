import React, { useMemo } from 'react';

import CheckpointModalTrigger from 'components/CheckpointModalTrigger';
import Grid, { GridMode } from 'components/Grid';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import TimeAgo from 'components/TimeAgo';
import TimeDuration from 'components/TimeDuration';
import { getDuration } from 'shared/utils/datetime';
import { humanReadableBytes } from 'shared/utils/string';
import { ShirtSize } from 'themes';
import { CheckpointWorkloadExtended, ExperimentBase, TrialDetails } from 'types';

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
  }, [trial]);

  const totalCheckpointsSize = useMemo(() => {
    const totalBytes = trial?.totalCheckpointSize || experiment.checkpointSize;
    if (!totalBytes) return;
    return humanReadableBytes(totalBytes);
  }, [trial, experiment]);

  const startTime = useMemo(() => {
    return trial?.startTime || experiment.startTime;
  }, [trial, experiment]);

  return (
    <Section>
      <Grid gap={ShirtSize.Medium} minItemWidth={180} mode={GridMode.AutoFill}>
        {trial?.runnerState && (
          <OverviewStats title="Last Runner State">{trial.runnerState}</OverviewStats>
        )}
        {startTime && (
          <OverviewStats title="Start Time">
            <TimeAgo datetime={startTime} />
          </OverviewStats>
        )}
        {!trial && experiment?.endTime && (
          <OverviewStats title="Duration">
            <TimeDuration duration={getDuration(experiment)} />
          </OverviewStats>
        )}
        {totalCheckpointsSize && (
          <OverviewStats title="Total Checkpoint Size">{totalCheckpointsSize}</OverviewStats>
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
