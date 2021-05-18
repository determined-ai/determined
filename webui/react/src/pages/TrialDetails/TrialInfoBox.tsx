import React, { useCallback, useMemo, useState } from 'react';

import CheckpointModal from 'components/CheckpointModal';
import InfoBox, { InfoboxStyle } from 'components/InfoBox';
import Section from 'components/Section';
import {
  CheckpointDetail, CheckpointState, CheckpointWorkload, ExperimentBase, TrialDetails,
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
      .filter(step => step.checkpoint
        && step.checkpoint.state === CheckpointState.Completed)
      .map(step =>checkpointSize(step.checkpoint as CheckpointWorkload))
      .reduce((acc, cur) => acc + cur, 0);
    return humanReadableBytes(totalBytes);
  }, [ trial.workloads ]);

  const durations = useMemo(() => trialDurations(trial.workloads), [ trial.workloads ]);

  const handleShowBestCheckpoint = useCallback(() => setShowBestCheckpoint(true), []);
  const handleHideBestCheckpoint = useCallback(() => setShowBestCheckpoint(false), []);

  const infoRows = [
    {
      content: shortEnglishHumannizer(getDuration({ startTime: trial.startTime })) + ' ago',
      label: 'Start Time',
    },
    {
      content: shortEnglishHumannizer(durations.train),
      label: 'Training Time',
    },
    {
      content: shortEnglishHumannizer(durations.validation),
      label: 'Validation Time',
    },
    {
      content: shortEnglishHumannizer(durations.checkpoint),
      label: 'Checkpointing Time',
    },
    {
      content: totalCheckpointsSize,
      label: 'Total Checkpoint Size',
    },
    {
      content: bestCheckpoint && <>Trial {bestCheckpoint.trialId} Batch {bestCheckpoint.batch}</>,
      label: 'Best Checkpoint',
      onClick: handleShowBestCheckpoint,
    },
  ];

  return (
    <Section>
      <InfoBox rows={infoRows} style={InfoboxStyle.Boxed} />
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
