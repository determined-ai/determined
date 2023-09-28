import { Modal, Tag, Typography } from 'antd';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import HumanReadableNumber from 'components/HumanReadableNumber';
import { XOR } from 'components/kit/internal/types';
import usePrevious from 'components/kit/internal/usePrevious';
import Select, { Option, SelectValue } from 'components/kit/Select';
import Spinner from 'components/kit/Spinner';
import { Loadable } from 'components/kit/utils/loadable';
import Link from 'components/Link';
import Message from 'components/Message';
import MetricBadgeTag from 'components/MetricBadgeTag';
import MetricSelect from 'components/MetricSelect';
import useMetricNames from 'hooks/useMetricNames';
import useResize from 'hooks/useResize';
import { paths } from 'routes/utils';
import { getTrialDetails } from 'services/api';
import { ExperimentItem, Metric, MetricSummary, Primitive, TrialDetails } from 'types';
import { isNumber } from 'utils/data';
import handleError, { ErrorType } from 'utils/error';
import { humanReadableBytes, pluralizer } from 'utils/string';

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
  trials,
  experiment,
  onUnselect,
}: TableProps) => {
  const [trialsDetails, setTrialsDetails] = useState(trials ?? []);
  const [selectedHyperparameters, setSelectedHyperparameters] = useState<string[]>([]);
  const [selectedMetrics, setSelectedMetrics] = useState<Metric[]>([]);
  const colSpan = Array.isArray(experiment) ? experiment.length + 1 : 1;

  useEffect(() => {
    if (trialIds === undefined) return;
    const canceler = new AbortController();

    const fetchTrialDetails = async (trialId: number) => {
      try {
        const response = await getTrialDetails({ id: trialId }, { signal: canceler.signal });
        setTrialsDetails((prev) => [...prev, response]);
      } catch (e) {
        handleError(e);
      }
    };

    setTrialsDetails([]);
    trialIds.forEach((trialId) => {
      fetchTrialDetails(trialId);
    });

    return () => {
      canceler.abort();
    };
  }, [trialIds]);

  useEffect(() => {
    if (trials === undefined) return;
    setTrialsDetails(trials);
  }, [trials]);

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

  const experimentMap = useMemo(() => {
    return Array.isArray(experiment)
      ? experiment.reduce(
          (acc, cur) => ({ ...acc, [cur.id]: cur }),
          {} as Record<number, ExperimentItem>,
        )
      : { [experiment.id]: experiment };
  }, [experiment]);

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
        )} ${experimentIds.join(', ')}.`,
        publicSubject: 'Experiment metric name stream failed.',
        type: ErrorType.Api,
      });
    },
    [experimentIds],
  );

  const loadableMetrics = useMetricNames(experimentIds, handleMetricNamesError);
  const metrics: Metric[] = useMemo(() => {
    return Loadable.getOrElse([], loadableMetrics);
  }, [loadableMetrics]);

  const prevMetrics = usePrevious(metrics, []);

  useEffect(() => {
    setSelectedMetrics((prevSelectedMetrics) =>
      _.isEqual(prevSelectedMetrics, prevMetrics) ? metrics : prevSelectedMetrics,
    );
  }, [metrics, prevMetrics]);

  const onMetricSelect = useCallback((selectedMetrics: Metric[]) => {
    setSelectedMetrics(selectedMetrics);
  }, []);

  const latestMetrics = useMemo(
    () =>
      trialsDetails.reduce((metricValues, trial) => {
        metricValues[trial.id] = Object.values<Record<string, MetricSummary> | null>(
          trial.summaryMetrics ?? {},
        ).reduce((trialMetrics, curMetricType) => {
          for (const [metricName, metricSummary] of Object.entries<MetricSummary>(
            curMetricType ?? {},
          )) {
            if (metricSummary.last != null) trialMetrics[metricName] = metricSummary.last;
          }
          return trialMetrics;
        }, {} as Record<string, Primitive>);
        return metricValues;
      }, {} as Record<number, Record<string, Primitive>>),
    [trialsDetails],
  );

  const hyperparameterNames = useMemo(() => {
    return [
      ...trialsDetails.reduce((hpSet, curTrial) => {
        Object.keys(curTrial.hyperparameters).forEach((hp) => hpSet.add(hp));
        return hpSet;
      }, new Set<string>()),
    ];
  }, [trialsDetails]);

  const prevHps = usePrevious(hyperparameterNames, []);

  useEffect(() => {
    setSelectedHyperparameters((prevSelectedHps) =>
      _.isEqual(prevSelectedHps, prevHps) ? hyperparameterNames : prevSelectedHps,
    );
  }, [hyperparameterNames, prevHps]);

  const onHyperparameterSelect = useCallback((selectedHPs: SelectValue) => {
    setSelectedHyperparameters(selectedHPs as string[]);
  }, []);

  const isLoaded = useMemo(
    () => (trialIds ? trialsDetails.length === trialIds.length : true),
    [trialIds, trialsDetails],
  );

  return (
    <div className={css.base}>
      {!(
        (trialIds === undefined || trialIds.length === 0) &&
        (trials === undefined || trials.length === 0)
      ) ? (
        <Spinner center spinning={!isLoaded}>
          <table>
            <thead>
              <tr>
                <th />
                {trialsDetails.map((trial) => (
                  <th key={trial.id}>
                    <Tag
                      className={css.trialTag}
                      closable={!!onUnselect}
                      style={{ width: '100%' }}
                      onClose={() => handleTrialUnselect(trial.id)}>
                      <Link path={paths.trialDetails(trial.id, trial.experimentId)}>
                        {Array.isArray(experiment) ? (
                          <Typography.Text ellipsis={{ tooltip: true }}>
                            {experimentMap[trial.experimentId]?.name}
                          </Typography.Text>
                        ) : (
                          `Trial ${trial.id}`
                        )}
                      </Link>
                    </Tag>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              <tr>
                <th scope="row">State</th>
                {trialsDetails.map((trial) => (
                  <td key={trial.id} style={{ textAlign: 'center' }}>
                    <Badge state={trial.state} type={BadgeType.State} />
                  </td>
                ))}
              </tr>
              {Array.isArray(experiment) && (
                <>
                  <tr>
                    <th scope="row">Experiment ID</th>
                    {trialsDetails.map((trial) => (
                      <td key={trial.id}>
                        <Typography.Text ellipsis={{ tooltip: true }}>
                          {trial.experimentId}
                        </Typography.Text>
                      </td>
                    ))}
                  </tr>
                  <tr>
                    <th scope="row">Trial ID</th>
                    {trialsDetails.map((trial) => (
                      <td key={trial.id}>
                        <Typography.Text ellipsis={{ tooltip: true }}>{trial.id}</Typography.Text>
                      </td>
                    ))}
                  </tr>
                </>
              )}
              <tr>
                <th scope="row">Batched Processed</th>
                {trialsDetails.map((trial) => (
                  <td key={trial.id}>
                    <Typography.Text ellipsis={{ tooltip: true }}>
                      {trial.totalBatchesProcessed}
                    </Typography.Text>
                  </td>
                ))}
              </tr>
              <tr>
                <th scope="row">Total Checkpoint Size</th>
                {trialsDetails.map((trial) => (
                  <td key={trial.id}>
                    <Typography.Text ellipsis={{ tooltip: true }}>
                      {totalCheckpointsSizes[trial.id]}
                    </Typography.Text>
                  </td>
                ))}
              </tr>
              <tr>
                <th className={css.tableSelectCell} colSpan={colSpan} scope="row">
                  <div className={css.tableSelectContainer}>
                    <MetricSelect
                      defaultMetrics={metrics}
                      metrics={metrics}
                      multiple
                      value={selectedMetrics}
                      onChange={onMetricSelect}
                    />
                  </div>
                </th>
              </tr>
              {selectedMetrics.map((metric) => (
                <tr key={`${metric.group}-${metric.name}`}>
                  <th scope="row">
                    <MetricBadgeTag metric={metric} />
                  </th>
                  {trialsDetails.map((trial) => {
                    const metricValue = latestMetrics[trial.id][metric.name];
                    return (
                      <td key={trial.id}>
                        {metricValue !== undefined ? (
                          typeof metricValue === 'number' ? (
                            <HumanReadableNumber num={metricValue} />
                          ) : (
                            metricValue
                          )
                        ) : null}
                      </td>
                    );
                  })}
                </tr>
              ))}
              <tr>
                <th className={css.tableSelectCell} colSpan={colSpan} scope="row">
                  <div className={css.tableSelectContainer}>
                    <Select
                      defaultValue={hyperparameterNames}
                      disableTags
                      label="Hyperparameters"
                      mode="multiple"
                      value={selectedHyperparameters}
                      width={200}
                      onChange={onHyperparameterSelect}>
                      {hyperparameterNames.map((hp) => (
                        <Option key={hp} value={hp}>
                          {hp}
                        </Option>
                      ))}
                    </Select>
                  </div>
                </th>
              </tr>
              {selectedHyperparameters.map((hp) => (
                <tr key={hp}>
                  <th scope="row">
                    <Typography.Text ellipsis={{ tooltip: true }}>{hp}</Typography.Text>
                  </th>
                  {trialsDetails.map((trial) => {
                    const hpValue = trial.hyperparameters[hp];
                    const stringValue = JSON.stringify(hpValue);
                    return (
                      <td key={trial.id}>
                        {isNumber(hpValue) ? (
                          <HumanReadableNumber num={hpValue} />
                        ) : (
                          <Typography.Text ellipsis={{ tooltip: true }}>
                            {stringValue}
                          </Typography.Text>
                        )}
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </Spinner>
      ) : (
        <Message title="No data available." />
      )}
    </div>
  );
};

export default TrialsComparisonModal;
