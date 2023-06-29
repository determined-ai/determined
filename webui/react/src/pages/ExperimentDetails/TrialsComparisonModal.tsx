import { Modal, Tag, Typography } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import HumanReadableNumber from 'components/HumanReadableNumber';
import Empty from 'components/kit/Empty';
import { isEqual } from 'components/kit/internal/functions';
import { XOR } from 'components/kit/internal/types';
import usePrevious from 'components/kit/internal/usePrevious';
import Select, { Option, SelectValue } from 'components/kit/Select';
import Link from 'components/Link';
import MetricBadgeTag from 'components/MetricBadgeTag';
import MetricSelect from 'components/MetricSelect';
import Spinner from 'components/Spinner/Spinner';
import useMetricNames from 'hooks/useMetricNames';
import useResize from 'hooks/useResize';
import { paths } from 'routes/utils';
import { getTrialDetails } from 'services/api';
import { ExperimentItem, Metric, MetricSummary, Primitive, TrialDetails } from 'types';
import { isNumber } from 'utils/data';
import handleError, { ErrorType } from 'utils/error';
import { Loadable } from 'utils/loadable';
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
  const metrics = Loadable.getOrElse([], loadableMetrics);

  const prevMetrics = usePrevious(metrics, []);

  useEffect(() => {
    setSelectedMetrics((prevSelectedMetrics) =>
      isEqual(prevSelectedMetrics, prevMetrics) ? metrics : prevSelectedMetrics,
    );
  }, [metrics, prevMetrics]);

  const onMetricSelect = useCallback((selectedMetrics: Metric[]) => {
    setSelectedMetrics(selectedMetrics);
  }, []);

  const latestMetrics = useMemo(
    () =>
      trialsDetails.reduce((metricValues, trial) => {
        metricValues[trial.id] = Object.values(trial.summaryMetrics ?? {}).reduce(
          (trialMetrics, curMetricType: Record<string, MetricSummary> | undefined) => {
            for (const [metricName, metricSummary] of Object.entries(curMetricType ?? {})) {
              trialMetrics[metricName] = metricSummary.last;
            }
            return trialMetrics;
          },
          {},
        );
        return metricValues;
      }, {} as Record<number, Record<string, Primitive | undefined>>),
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
      isEqual(prevSelectedHps, prevHps) ? hyperparameterNames : prevSelectedHps,
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
          <div className={[css.row, css.sticky].join(' ')}>
            <div className={[css.cell, css.blank, css.sticky].join(' ')} />
            {trialsDetails.map((trial) => (
              <div className={css.cell} key={trial.id}>
                <Tag
                  className={css.trialTag}
                  closable={!!onUnselect}
                  onClose={() => handleTrialUnselect(trial.id)}>
                  <Link path={paths.trialDetails(trial.id, trial.experimentId)}>
                    {Array.isArray(experiment) ? (
                      <Typography.Paragraph ellipsis={{ tooltip: true }}>
                        {experimentMap[trial.experimentId]?.name}
                      </Typography.Paragraph>
                    ) : (
                      `Trial ${trial.id}`
                    )}
                  </Link>
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
          {Array.isArray(experiment) && (
            <>
              <div className={css.row}>
                <div className={[css.cell, css.sticky, css.indent].join(' ')}>Experiment ID</div>
                {trialsDetails.map((trial) => (
                  <div className={css.cell} key={trial.id}>
                    {trial.experimentId}
                  </div>
                ))}
              </div>
              <div className={css.row}>
                <div className={[css.cell, css.sticky, css.indent].join(' ')}>Trial ID</div>
                {trialsDetails.map((trial) => (
                  <div className={css.cell} key={trial.id}>
                    {trial.id}
                  </div>
                ))}
              </div>
            </>
          )}
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
              <MetricSelect
                defaultMetrics={metrics}
                metrics={metrics}
                multiple
                value={selectedMetrics}
                onChange={onMetricSelect}
              />
            </div>
          </div>
          {selectedMetrics.map((metric) => (
            <div className={css.row} key={metric.name}>
              <div className={[css.cell, css.sticky, css.indent].join(' ')}>
                <MetricBadgeTag metric={metric} />
              </div>
              {trialsDetails.map((trial) => {
                const metricValue = latestMetrics[trial.id][metric.name];
                return (
                  <div className={css.cell} key={trial.id}>
                    {metricValue !== undefined ? (
                      typeof metricValue === 'number' ? (
                        <HumanReadableNumber num={metricValue} />
                      ) : (
                        metricValue
                      )
                    ) : (
                      ''
                    )}
                  </div>
                );
              })}
            </div>
          ))}
          <div className={[css.row, css.spanAll].join(' ')}>
            <div className={[css.cell, css.spanAll].join(' ')}>
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
          </div>
          {selectedHyperparameters.map((hp) => (
            <div className={css.row} key={hp}>
              <div className={[css.cell, css.sticky, css.indent].join(' ')}>
                <Typography.Paragraph ellipsis={{ tooltip: true }}>{hp}</Typography.Paragraph>
              </div>
              {trialsDetails.map((trial) => {
                const hpValue = trial.hyperparameters[hp];
                const stringValue = JSON.stringify(hpValue);
                return (
                  <div className={css.cell} key={trial.id}>
                    {isNumber(hpValue) ? (
                      <HumanReadableNumber num={hpValue} />
                    ) : (
                      <Typography.Paragraph ellipsis={{ tooltip: true }}>
                        {stringValue}
                      </Typography.Paragraph>
                    )}
                  </div>
                );
              })}
            </div>
          ))}
        </Spinner>
      ) : (
        <Empty
          icon="document"
          title={`No ${Array.isArray(experiment) ? 'experiments with ' : ''}trials selected`}
        />
      )}
    </div>
  );
};

export default TrialsComparisonModal;
