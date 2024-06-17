import Card from 'hew/Card';
import Column from 'hew/Column';
import { Modal, useModal } from 'hew/Modal';
import Row from 'hew/Row';
import { isEqual, sumBy, uniq } from 'lodash';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Link from 'components/Link';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import TimeAgo from 'components/TimeAgo';
import { pausableRunStates } from 'constants/states';
import { useCheckpointFlow } from 'hooks/useCheckpointFlow';
import usePolling from 'hooks/usePolling';
import { NodeElement } from 'pages/ResourcePool/Topology';
import { paths } from 'routes/utils';
import { getTaskAcceleratorData } from 'services/api';
import { V1AcceleratorData } from 'services/api-ts-sdk/api';
import { CheckpointWorkloadExtended, ExperimentBase, TrialDetails } from 'types';
import handleError from 'utils/error';
import { humanReadableBytes } from 'utils/string';

import css from './TrialInfoBox.module.scss';

const allocationModalComponent: React.FC<{ data?: V1AcceleratorData[] }> = ({ data }) => {
  return (
    <Modal size="medium" title="Resource Allocation">
      {data && (
        <>
          <Row wrap>
            {data.map((d) => (
              <NodeElement
                key={d.nodeName}
                name={d.nodeName || ''}
                numOfSlots={d.acceleratorUuids?.length || 1}
              />
            ))}
          </Row>
          <Column>
            <div className={css.dataRow}>
              <span>Resource Pool</span>
              <span className={css.dashline} />
              <span className={css.pools}>
                {uniq(data.map((d) => d.resourcePool))
                  .filter((r) => !!r)
                  .map((r) => (
                    <Link key={r} path={paths.resourcePool(r!)}>
                      {r}
                    </Link>
                  ))}
              </span>
            </div>
            <div className={css.dataRow}>
              <span>Accelerator</span>
              <span className={css.dashline} />
              <span>{uniq(data.map((d) => d.acceleratorType)).join(',')}</span>
            </div>
          </Column>
          {data.map((d) => (
            <Column key={d.nodeName}>
              <div className={css.dataRow}>
                <span>Node ID</span>
                <span className={css.dashline} />
                <span>{d.nodeName}</span>
              </div>
              {d.acceleratorUuids?.map((a, idx) => (
                <div className={css.dataRow} key={idx}>
                  <span>{`Slot ${idx + 1} ID`}</span>
                  <span className={css.dashline} />
                  <span>{a}</span>
                </div>
              ))}
            </Column>
          ))}
        </>
      )}
    </Modal>
  );
};

interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

const TrialInfoBox: React.FC<Props> = ({ trial, experiment }: Props) => {
  const [canceler] = useState(new AbortController());
  const [acceleratorData, setAcceleratorData] = useState<V1AcceleratorData[]>();

  const fetchAcceleratorData = useCallback(async () => {
    if (!trial?.taskId) return;
    try {
      const data = await getTaskAcceleratorData(
        { taskId: trial.taskId },
        { signal: canceler.signal },
      );
      setAcceleratorData((prev) => (isEqual(data, prev) ? prev : data));
    } catch (e) {
      handleError(e);
    }
  }, [trial, canceler]);

  const { stopPolling } = usePolling(fetchAcceleratorData, { rerunOnNewFn: true });

  useEffect(() => {
    return () => {
      canceler.abort();
      stopPolling();
    };
  }, [canceler, stopPolling]);

  useEffect(() => {
    if (acceleratorData && !pausableRunStates.has(experiment.state)) stopPolling();
  }, [experiment.state, acceleratorData, stopPolling]);

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

  const numOfSlots = useMemo(() => {
    return sumBy(acceleratorData, (d) => d.acceleratorUuids?.length || 0);
  }, [acceleratorData]);

  const allocationModal = useModal(allocationModalComponent);

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
        {acceleratorData && (
          <OverviewStats title="Resource Allocation" onClick={allocationModal.open}>
            <div
              style={{
                color: pausableRunStates.has(experiment.state)
                  ? 'none'
                  : 'var(--theme-status-active)',
              }}>
              {`${numOfSlots} Slots`}
            </div>
          </OverviewStats>
        )}
        {<OverviewStats title="Log Retention Days">{logRetentionDays}</OverviewStats>}
      </Card.Group>
      <allocationModal.Component data={acceleratorData} />
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
