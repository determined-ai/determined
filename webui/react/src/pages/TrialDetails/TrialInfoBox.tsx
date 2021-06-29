import React, { useCallback, useMemo, useState } from 'react';

import CheckpointModal from 'components/CheckpointModal';
import Grid, { GridMode } from 'components/Grid';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import { ShirtSize } from 'themes';
import {
  CheckpointDetail, CheckpointWorkload, ExperimentBase, TrialDetails,
} from 'types';
import { humanReadableBytes } from 'utils/string';
import { getDuration, shortEnglishHumannizer } from 'utils/time';
import { trialDurations } from 'utils/trial';
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
      .filter(wlWrapper => !!wlWrapper.checkpoint)
      .map(wlWrapper => checkpointSize(wlWrapper.checkpoint as CheckpointWorkload))
      .reduce((acc, cur) => acc + cur, 0);
    return humanReadableBytes(totalBytes);
  }, [ trial.workloads ]);

  const durations = useMemo(() => trialDurations(trial.workloads), [ trial.workloads ]);

  const handleShowBestCheckpoint = useCallback(() => setShowBestCheckpoint(true), []);
  const handleHideBestCheckpoint = useCallback(() => setShowBestCheckpoint(false), []);

  return (
    <Section>
      <Grid gap={ShirtSize.medium} minItemWidth={180} mode={GridMode.AutoFill}>
        <OverviewStats title="Start Time">
          {shortEnglishHumannizer(getDuration({ startTime: trial.startTime }))} ago
        </OverviewStats>
        <OverviewStats title="Training Time">
          {shortEnglishHumannizer(durations.train)}
        </OverviewStats>
        <OverviewStats title="Validation Time">
          {shortEnglishHumannizer(durations.validation)}
        </OverviewStats>
        <OverviewStats title="Checkpointing Time">
          {shortEnglishHumannizer(durations.checkpoint)}
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
