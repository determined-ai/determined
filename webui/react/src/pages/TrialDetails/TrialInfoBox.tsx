import Card from 'hew/Card';
import React, { useCallback, useMemo } from 'react';

import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import TimeAgo from 'components/TimeAgo';
import { useCheckpointFlow } from 'hooks/useCheckpointFlow';
import { handlePath } from 'routes/utils';
import { CheckpointWorkloadExtended, ExperimentBase, TrialDetails } from 'types';
import { createPachydermLineageLink } from 'utils/integrations';
import { AnyMouseEvent } from 'utils/routes';
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
      case 1:
        return `${trial?.logRetentionDays} day`;
      default:
        return `${trial?.logRetentionDays} days`;
    }
  }, [trial]);
  const integrationData = useMemo(() => {
    const {
      config: { integrations },
    } = experiment;

    if (integrations) {
      const url = createPachydermLineageLink(integrations);

      return {
        hasIntegrationData: url !== undefined ? true : false,
        text: '<MLDM repo>',
        url,
      };
    }

    return;
  }, [experiment]);

  const handleClickDataInput = useCallback(
    (e: AnyMouseEvent) => {
      if (integrationData?.hasIntegrationData)
        handlePath(e, {
          path: integrationData.url,
        });
    },
    [integrationData],
  );

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
        {<OverviewStats title="Log Retention Days">{logRetentionDays}</OverviewStats>}
        {integrationData && (
          <OverviewStats title="Data input" onClick={handleClickDataInput}>
            {integrationData.text}
          </OverviewStats>
        )}
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
