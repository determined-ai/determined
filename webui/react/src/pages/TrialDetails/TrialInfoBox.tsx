import Card from 'hew/Card';
import React, { useMemo } from 'react';

import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import TimeAgo from 'components/TimeAgo';
import { useCheckpointFlow } from 'hooks/useCheckpointFlow';
import { CheckpointWorkloadExtended, ExperimentBase, TrialDetails } from 'types';
import { humanReadableBytes } from 'utils/string';

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

  const { openCheckpoint, checkpointModalComponents } = useCheckpointFlow({
    checkpoint: bestCheckpoint,
    config: experiment.config,
    title: `Best checkpoint for Trial ${trial?.id}`,
  });

  const totalCheckpointsSize = useMemo(() => {
    const totalBytes = trial?.totalCheckpointSize;
    if (!totalBytes) return;
    return humanReadableBytes(totalBytes);
  }, [trial?.totalCheckpointSize]);

  const logRetentionDays = useMemo(() => {
    switch (trial?.logRetentionDays) {
      case undefined:
        return '-';
      case -1:
        return 'Forever';
      default:
        return `${trial?.logRetentionDays} days`;
    }
  }, [trial]);

  return (
    <Section>
      <Card.Group size="small">
        {trial?.runnerState && (
          <OverviewStats title="Last Runner State">{trial.runnerState}</OverviewStats>
        )}
        {trial?.startTime && (
          <OverviewStats title="Started">
            <TimeAgo datetime={trial.startTime} />
          </OverviewStats>
        )}
        {totalCheckpointsSize && (
          <OverviewStats title="Checkpoints">{`${trial?.checkpointCount} (${totalCheckpointsSize})`}</OverviewStats>
        )}
        {bestCheckpoint && (
          <>
            <OverviewStats title="Best Checkpoint" onClick={openCheckpoint}>
              Batch {bestCheckpoint.totalBatches}
            </OverviewStats>
            {checkpointModalComponents}
          </>
        )}
        {
          <OverviewStats data-testid="log-retention-box" title="Log Retention Days">
            {logRetentionDays}
          </OverviewStats>
        }
      </Card.Group>
    </Section>
  );
};

export default TrialInfoBox;

export const TrialInfoBoxMultiTrial: React.FC<Props> = ({ experiment }: Props) => {
  const searcher = experiment.config.searcher;
  const checkpointsSize = useMemo(() => {
    const totalBytes = experiment?.checkpointSize;
    if (!totalBytes) return;
    return humanReadableBytes(totalBytes);
  }, [experiment]);
  return (
    <Section>
      <Card.Group size="small">
        {searcher?.metric && <OverviewStats title="Metric">{searcher.metric}</OverviewStats>}
        {searcher?.name && <OverviewStats title="Searcher">{searcher.name}</OverviewStats>}
        {experiment.numTrials > 0 && (
          <OverviewStats title="Trials">{experiment.numTrials}</OverviewStats>
        )}
        {checkpointsSize && (
          <OverviewStats title="Checkpoints">
            {`${experiment.checkpoints} (${checkpointsSize})`}
          </OverviewStats>
        )}
      </Card.Group>
    </Section>
  );
};
