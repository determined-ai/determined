import { Modal, Tag } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import HumanReadableNumber from 'components/HumanReadableNumber';
import { XOR } from 'components/kit/internal/types';
import Select, { Option, SelectValue } from 'components/kit/Select';
import Tooltip from 'components/kit/Tooltip';
import Link from 'components/Link';
import MetricBadgeTag from 'components/MetricBadgeTag';
import MetricSelect from 'components/MetricSelect';
import Spinner from 'components/Spinner/Spinner';
import useMetricNames from 'hooks/useMetricNames';
import useResize from 'hooks/useResize';
import { paths } from 'routes/utils';
import { getTrialDetails, getTrialWorkloads } from 'services/api';
import Spinner from 'shared/components/Spinner/Spinner';
import { isNumber } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { humanReadableBytes, pluralizer } from 'shared/utils/string';
import { ExperimentItem, Metric, MetricsWorkload, TrialDetails, TrialWorkloadFilter } from 'types';
import handleError from 'utils/error';
import { humanReadableBytes } from 'utils/string';

import css from './TrialsComparisonModal.module.scss';

interface TablePropsBase {
  experiment: ExperimentItem | ExperimentItem[];
  onUnselect?: (trialId: number) => void;
}

type TableProps = XOR<
  {
    trialIds: number[];
  },
  {
    trials: TrialDetails[];
  }
> &
  TablePropsBase;

type ModalProps = TableProps & {
  onCancel: () => void;
  visible: boolean;
};

const TrialsComparisonModal: React.FC<ModalProps> = ({
  onCancel,
  visible,
  ...props
}: ModalProps) => {
  const resize = useResize();

  useEffect(() => {
    if (props.trialIds?.length === 0 || props.trials?.length === 0) onCancel();
  }, [onCancel, props.trialIds?.length, props.trials?.length]);

  return (
    <Modal
      centered
      footer={null}
      open={visible}
      style={{ height: resize.height * 0.9 }}
      title={
        !Array.isArray(props.experiment)
          ? `Experiment ${props.experiment.id} Trial Comparison`
          : 'Trial Comparison'
      }
      width={resize.width * 0.9}
      onCancel={onCancel}>
      <TrialsComparisonTable {...props} />
    </Modal>
  );
};

export const TrialsComparisonTable: React.FC<TableProps> = ({
  trialIds,
  trials = [],
  experiment,
  onUnselect,
}: TableProps) => {
  const [trialsDetails, setTrialsDetails] = useState(trials);
  const [canceler] = useState(new AbortController());
  const [selectedHyperparameters, setSelectedHyperparameters] = useState<string[]>([]);
  const [selectedMetrics, setSelectedMetrics] = useState<Metric[]>([]);

  const fetchTrialDetails = useCallback(
    async (trialId: number) => {
      try {
        const response = await getTrialDetails({ id: trialId }, { signal: canceler.signal });
        setTrialsDetails((prev) => [...prev, response]);
      } catch (e) {
        handleError(e);
      }
    },
    [canceler.signal],
  );

  useEffect(() => {
    return () => {
      canceler.abort();
    };
  }, [canceler]);

  useEffect(() => {
    if (!trialIds) return;
    trialIds.forEach((trial) => {
      fetchTrialDetails(trial);
    });
  }, [fetchTrialDetails, trialIds]);

  const handleTrialUnselect = useCallback((trialId: number) => onUnselect?.(trialId), [onUnselect]);

  const getCheckpointSize = useCallback((trial: TrialDetails) => {
    const totalBytes = trial.totalCheckpointSize;
    return humanReadableBytes(totalBytes);
  }, []);

  const totalCheckpointsSizes: Record<string, string> = useMemo(
    () =>
      Object.fromEntries(
        Object.values(trialsDetails).map((trial) => [trial.id, getCheckpointSize(trial)]),
      ),
    [getCheckpointSize, trialsDetails],
  );

  const experimentIds = useMemo(
    () => (Array.isArray(experiment) ? experiment.map((exp) => exp.id) : [experiment.id]),
    [experiment],
  );

  const handleMetricNamesError = useCallback(
    (e: unknown) => {
      handleError(e, {
        publicMessage: `Failed to load metric names for ${pluralizer(
          experimentIds.length,
          'experiment',
        )} {experimentIds.join(', ')}.`,
        publicSubject: 'Experiment metric name stream failed.',
        type: ErrorType.Api,
      });
    },
    [experimentIds],
  );

  const metrics = useMetricNames(experimentIds, handleMetricNamesError);

  useEffect(() => {
    setSelectedMetrics(metrics);
  }, [metrics]);

  const onMetricSelect = useCallback((selectedMetrics: Metric[]) => {
    setSelectedMetrics(selectedMetrics);
  }, []);

  const extractLatestMetrics = useCallback(
    (
      metricsObj: Record<number, Record<string, MetricsWorkload>>,
      workload: MetricsWorkload,
      trialId: number,
    ) => {
      for (const metricName of Object.keys(workload.metrics || {})) {
        if (metricsObj[trialId][metricName]) {
          if (
            new Date(workload.endTime || Date()).getTime() -
              new Date(metricsObj[trialId][metricName].endTime || Date()).getTime() >
            0
          ) {
            metricsObj[trialId][metricName] = workload;
          }
        } else {
          metricsObj[trialId][metricName] = workload;
        }
      }
      return metricsObj;
    },
    [],
  );

  const [latestMetrics, setLatestMetrics] = useState<Record<string, Record<string, number>>>({});

  useMemo(async () => {
    const metricsObj: Record<string, Record<string, MetricsWorkload>> = {};
    for (const trial of Object.values(trialsDetails)) {
      metricsObj[trial.id] = {};
      const data = await getTrialWorkloads({
        filter: TrialWorkloadFilter.All,
        id: trial.id,
        limit: 50,
        orderBy: 'ORDER_BY_DESC',
      });
      const latestWorkloads = data.workloads;
      latestWorkloads.forEach((workload) => {
        if (workload.training) {
          extractLatestMetrics(metricsObj, workload.training, trial.id);
        } else if (workload.validation) {
          extractLatestMetrics(metricsObj, workload.validation, trial.id);
        }
      });
    }
    const metricValues: Record<number, Record<string, number>> = {};
    for (const [trialId, metrics] of Object.entries(metricsObj)) {
      metricValues[Number(trialId)] = {};
      for (const [metric, workload] of Object.entries(metrics)) {
        if (workload.metrics) {
          metricValues[Number(trialId)][metric] = workload?.metrics[metric];
        }
      }
    }
    setLatestMetrics(metricValues);
  }, [extractLatestMetrics, trialsDetails]);

  const hyperparameterNames = useMemo(
    () => Object.keys(trialsDetails.first()?.hyperparameters || {}),
    [trialsDetails],
  );

  useEffect(() => {
    setSelectedHyperparameters(hyperparameterNames);
  }, [hyperparameterNames]);

  const onHyperparameterSelect = useCallback((selectedHPs: SelectValue) => {
    setSelectedHyperparameters(selectedHPs as string[]);
  }, []);

  const isLoaded = useMemo(
    () => (trialIds ? trialsDetails.length === trialIds.length : true),
    [trialIds, trialsDetails],
  );

  return (
    <div className={css.base}>
      {isLoaded ? (
        <>
          <div className={[css.row, css.sticky].join(' ')}>
            <div className={[css.cell, css.blank, css.sticky].join(' ')} />
            {trialsDetails.map((trial) => (
              <div className={css.cell} key={trial.id}>
                <Tag
                  className={css.trialTag}
                  closable={!!onUnselect}
                  onClose={() => handleTrialUnselect(trial.id)}>
                  <Link path={paths.trialDetails(trial.id)}>Trial {trial.id}</Link>
                </Tag>
              </div>
            ))}
          </div>
          <div className={css.row}>
            <div className={[css.cell, css.sticky, css.indent].join(' ')}>State</div>
            {trialsDetails.map((trial) => (
              <div className={css.cell} key={trial.id}>
                <Badge state={trial.state} type={BadgeType.State} />
              </div>
            ))}
          </div>
          <div className={css.row}>
            <div className={[css.cell, css.sticky, css.indent].join(' ')}>Batched Processed</div>
            {trialsDetails.map((trial) => (
              <div className={css.cell} key={trial.id}>
                {trial.totalBatchesProcessed}
              </div>
            ))}
          </div>
          <div className={css.row}>
            <div className={[css.cell, css.sticky, css.indent].join(' ')}>
              Total Checkpoint Size
            </div>
            {trialsDetails.map((trial) => (
              <div className={css.cell} key={trial.id}>
                {totalCheckpointsSizes[trial.id]}
              </div>
            ))}
          </div>
          <div className={[css.row, css.spanAll].join(' ')}>
            <div className={[css.cell, css.spanAll].join(' ')}>
              Metrics
              <MetricSelect
                defaultMetrics={metrics}
                label=""
                metrics={metrics}
                multiple
                value={selectedMetrics}
                onChange={onMetricSelect}
              />
            </div>
          </div>
          {metrics
            .filter((metric) => selectedMetrics.map((m) => m.name).includes(metric.name))
            .map((metric) => (
              <div className={css.row} key={metric.name}>
                <div className={[css.cell, css.sticky, css.indent].join(' ')}>
                  <MetricBadgeTag metric={metric} />
                </div>
                {trialsDetails.map((trial) => (
                  <div className={css.cell} key={trial.id}>
                    {latestMetrics[trial.id] ? (
                      <HumanReadableNumber num={latestMetrics[trial.id][metric.name] || 0} />
                    ) : (
                      ''
                    )}
                  </div>
                ))}
              </div>
            ))}
          <div className={[css.row, css.spanAll].join(' ')}>
            <div className={[css.cell, css.spanAll].join(' ')}>
              Hyperparameters
              <Select
                disableTags
                label=""
                mode="multiple"
                value={selectedHyperparameters}
                onChange={onHyperparameterSelect}>
                {hyperparameterNames.map((hp) => (
                  <Option key={hp} value={hp}>
                    {hp}
                  </Option>
                ))}
              </Select>
            </div>
          </div>
          {selectedHyperparameters.map((hp) => (
            <div className={css.row} key={hp}>
              <div className={[css.cell, css.sticky, css.indent].join(' ')}>{hp}</div>
              {trialsDetails.map((trial) => {
                const hpValue = trial.hyperparameters[hp];
                const stringValue = JSON.stringify(hpValue);
                return (
                  <div className={css.cell} key={trial.id}>
                    {isNumber(hpValue) ? (
                      <HumanReadableNumber num={hpValue} />
                    ) : (
                      <Tooltip content={stringValue}>{stringValue}</Tooltip>
                    )}
                  </div>
                );
              })}
            </div>
          ))}
        </>
      ) : (
        <Spinner center spinning={!isLoaded} />
      )}
    </div>
  );
};

export default TrialsComparisonModal;
