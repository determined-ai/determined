import React, { useCallback, useMemo, useState } from 'react';

import CheckpointModal from 'components/CheckpointModal';
import Grid, { GridMode } from 'components/Grid';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import { ShirtSize } from 'themes';
import {
  CheckpointDetail, CheckpointState, CheckpointWorkload, ExperimentBase, TrialDetails,
} from 'types';
import { humanReadableBytes } from 'utils/string';
import { getDuration, shortEnglishHumannizer } from 'utils/time';
import { checkpointSize } from 'utils/types';

interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

const TrialInfoBox: React.FC<Props> = ({ trial, experiment }: Props) => {
  const [ showBestCheckpoint, setShowBestCheckpoint ] = useState(false);

  const bestCheckpoint: CheckpointDetail | undefined = useMemo(() => {
    const cp = trial.bestAvailableCheckpoint;
    if (!cp) return;

    return {
      ...cp,
      batch: cp.totalBatches,
      experimentId: trial.experimentId,
      trialId: trial.id,
    };
  }, [ trial.bestAvailableCheckpoint, trial.experimentId, trial.id ]);

  const totalCheckpointsSize = useMemo(() => {
    const totalBytes = trial.workloads
      .filter(step => step.checkpoint
        && step.checkpoint.state === CheckpointState.Completed)
      .map(step => checkpointSize(step.checkpoint as CheckpointWorkload))
      .reduce((acc, cur) => acc + cur, 0);
    return humanReadableBytes(totalBytes);
  }, [ trial.workloads ]);

  const handleShowBestCheckpoint = useCallback(() => setShowBestCheckpoint(true), []);
  const handleHideBestCheckpoint = useCallback(() => setShowBestCheckpoint(false), []);

  return (
    <Section>
      <Grid gap={ShirtSize.medium} minItemWidth={180} mode={GridMode.AutoFill}>
        {trial.runnerState &&
          <OverviewStats title="Last Runner State">
            {trial.runnerState}
          </OverviewStats>
        }
        <OverviewStats title="Start Time">
          {shortEnglishHumannizer(getDuration({ startTime: trial.startTime }))} ago
        </OverviewStats>
        <OverviewStats title="Total Checkpoint Size">
          {totalCheckpointsSize}
        </OverviewStats>
        {bestCheckpoint && (
          <OverviewStats title="Best Checkpoint" onClick={handleShowBestCheckpoint}>
            Batch {bestCheckpoint.batch}
          </OverviewStats>
        )}
      </Grid>

      {bestCheckpoint && (
        <CheckpointModal
          checkpoint={bestCheckpoint}
          config={experiment.config}
          show={showBestCheckpoint}
          title={`Best Checkpoint for Trial ${trial.id}`}
          onHide={handleHideBestCheckpoint} />
      )}
    </Section>
  );
};

export default TrialInfoBox;
