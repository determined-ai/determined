import { Select, Tag, Tooltip } from 'antd';
import Modal from 'antd/lib/modal/Modal';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import HumanReadableNumber from 'components/HumanReadableNumber';
import Link from 'components/Link';
import MetricBadgeTag from 'components/MetricBadgeTag';
import MetricSelectFilter from 'components/MetricSelectFilter';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import useResize from 'hooks/useResize';
import { paths } from 'routes/utils';
import { getTrialDetails } from 'services/api';
import { isNumber } from 'shared/utils/data';
import { humanReadableBytes } from 'shared/utils/string';
import {
  CheckpointState, CheckpointWorkload, ExperimentBase, MetricName, MetricsWorkload, TrialDetails,
} from 'types';
import handleError from 'utils/error';
import { extractMetricNames } from 'utils/metric';
import { checkpointSize } from 'utils/workload';

import css from './TrialsComparisonModal.module.scss';

const { Option } = Select;

interface ModalProps {
  experiment: ExperimentBase;
  onCancel: () => void;
  onUnselect: (trialId: number) => void;
  trials: number[];
  visible: boolean;
}

interface TableProps {
  experiment: ExperimentBase;
  onUnselect: (trialId: number) => void;
  trials: number[];
}

const TrialsComparisonModal: React.FC<ModalProps> =
({ experiment, onCancel, onUnselect, trials, visible }: ModalProps) => {
  const resize = useResize();

  useEffect(() => {
    if (trials.length === 0) onCancel();
  }, [ trials, onCancel ]);

  return (
    <Modal
      centered
      footer={null}
      style={{ height: resize.height * .9 }}
      title={`Experiment ${experiment.id} Trial Comparison`}
      visible={visible}
      width={resize.width * .9}
      onCancel={onCancel}>
      <TrialsComparisonTable experiment={experiment} trials={trials} onUnselect={onUnselect} />
    </Modal>
  );
};

const TrialsComparisonTable: React.FC<TableProps> = (
  { trials, experiment, onUnselect }: TableProps,
) => {
  const [ trialsDetails, setTrialsDetails ] = useState<Record<string, TrialDetails>>({});
  const [ canceler ] = useState(new AbortController());
  const [ selectedHyperparameters, setSelectedHyperparameters ] = useState<string[]>([]);
  const [ selectedMetrics, setSelectedMetrics ] = useState<MetricName[]>([]);

  const fetchTrialDetails = useCallback(async (trialId) => {
    try {
      const response = await getTrialDetails({ id: trialId }, { signal: canceler.signal });
      setTrialsDetails(prev => ({ ...prev, [trialId]: response }));
    } catch (e) {
      handleError(e);
    }
  }, [ canceler.signal ]);

  useEffect(() => {
    return () => {
      canceler.abort();
    };
  }, [ canceler ]);

  useEffect(() => {
    trials.forEach(trial => {
      fetchTrialDetails(trial);
    });
  }, [ fetchTrialDetails, trials ]);

  const handleTrialUnselect = useCallback((trialId: number) => onUnselect(trialId), [ onUnselect ]);

  const getCheckpointSize = useCallback((trial: TrialDetails) => {
    const totalBytes = trial.workloads
      .filter(step => step.checkpoint
      && step.checkpoint.state === CheckpointState.Completed)
      .map(step => checkpointSize(step.checkpoint as CheckpointWorkload))
      .reduce((acc, cur) => acc + cur, 0);
    return humanReadableBytes(totalBytes);
  }, []);

  const totalCheckpointsSizes: Record<string, string> = useMemo(
    () => Object.fromEntries(Object.values(trialsDetails)
      .map(trial => [ trial.id, getCheckpointSize(trial) ]))
    , [ getCheckpointSize, trialsDetails ],
  );

  const metricNames = useMemo(() => {
    const nameSet: Record<string, MetricName> = {};
    trials.forEach(trial => {
      extractMetricNames(trialsDetails[trial]?.workloads || [])
        .forEach(item => nameSet[item.name] = (item));
    });
    return Object.values(nameSet);
  }, [ trialsDetails, trials ]);

  useEffect(() => {
    setSelectedMetrics(metricNames);
  }, [ metricNames ]);

  const onMetricSelect = useCallback((selectedMetrics: MetricName[]) => {
    setSelectedMetrics(selectedMetrics);
  }, []);

  const extractLatestMetrics = useCallback((
    metricsObj: Record<string, {[key: string]: MetricsWorkload}>,
    workload: MetricsWorkload,
    trialId: number,
  ) => {
    for (const metricName of
      Object.keys(workload.metrics || {})) {
      if (metricsObj[trialId][metricName]) {
        if ((new Date(workload.endTime || Date())).getTime() -
        (new Date(metricsObj[trialId][metricName].endTime || Date()).getTime()) > 0) {
          metricsObj[trialId][metricName] = workload;
        }
      } else {
        metricsObj[trialId][metricName] = workload;
      }
    }
    return metricsObj;
  }, []);

  const metrics = useMemo(() => {
    const metricsObj: Record<string, {[key: string]: MetricsWorkload}> = {};
    for (const trial of Object.values(trialsDetails)) {
      metricsObj[trial.id] = {};
      trial.workloads.forEach(workload => {
        if (workload.training) {
          extractLatestMetrics(metricsObj, workload.training, trial.id);
        } else if (workload.validation) {
          extractLatestMetrics(metricsObj, workload.validation, trial.id);
        }
      });
    }
    const metricValues: Record<string, {[key: string]: number}> = {};
    for (const [ trialId, metrics ] of Object.entries(metricsObj)) {
      metricValues[trialId] = {};
      for (const [ metric, workload ] of Object.entries(metrics)) {
        if (workload.metrics){
          metricValues[trialId][metric] = workload.metrics[metric];
        }
      }
    }
    return metricValues;
  }, [ extractLatestMetrics, trialsDetails ]);

  const hyperparameterNames = useMemo(
    () => Object.keys(trialsDetails[trials.first()]?.hyperparameters || {}),
    [ trials, trialsDetails ],
  );

  useEffect(() => {
    setSelectedHyperparameters(hyperparameterNames);
  }, [ hyperparameterNames ]);

  const onHyperparameterSelect = useCallback((selectedHPs) => {
    setSelectedHyperparameters(selectedHPs);
  }, []);

  const isLoaded = useMemo(
    () => trials.every(trialId => trialsDetails[trialId])
    , [ trials, trialsDetails ],
  );

  return (
    <div className={css.base}>
      {isLoaded ? (
        <>
          <div className={[ css.row, css.header, css.sticky ].join(' ')}>
            <div className={[ css.cell, css.blank, css.header, css.sticky ].join(' ')} />
            {trials.map(trialId => (
              <div className={css.cell} key={trialId}>
                <Tag
                  className={css.trialTag}
                  closable
                  onClose={() => handleTrialUnselect(trialId)}>
                  <Link path={paths.trialDetails(trialId, experiment.id)}>Trial {trialId}</Link>
                </Tag>
              </div>
            ))}
          </div>
          <div className={css.row}>
            <div className={[ css.cell, css.header, css.sticky, css.indent ].join(' ')}>State</div>
            {trials.map(trial => (
              <div className={css.cell} key={trial}>
                <Badge state={trialsDetails[trial].state} type={BadgeType.State} />
              </div>
            ))}
          </div>
          <div className={css.row}>
            <div className={[ css.cell, css.header, css.sticky, css.indent ].join(' ')}>
              Batched Processed
            </div>
            {trials.map(trialId => (
              <div className={css.cell} key={trialId}>
                {trialsDetails[trialId].totalBatchesProcessed}
              </div>
            ))}
          </div>
          <div className={css.row}>
            <div className={[ css.cell, css.header, css.sticky, css.indent ].join(' ')}>
              Total Checkpoint Size
            </div>
            {trials.map(trialId => (
              <div className={css.cell} key={trialId}>{totalCheckpointsSizes[trialId]}</div>
            ))}
          </div>
          <div className={[ css.row, css.header, css.spanAll ].join(' ')}>
            <div className={[ css.cell, css.header, css.spanAll ].join(' ')}>
              Metrics
              <MetricSelectFilter
                defaultMetricNames={metricNames}
                label=""
                metricNames={metricNames}
                multiple
                value={selectedMetrics}
                onChange={onMetricSelect}
              />
            </div>
          </div>
          {metricNames.filter(metric => selectedMetrics
            .map(m => m.name)
            .includes(metric.name))
            .map(metric => (
              <div className={css.row} key={metric.name}>
                <div className={[ css.cell, css.header, css.sticky, css.indent ].join(' ')}>
                  <MetricBadgeTag metric={metric} />
                </div>
                {trials.map(trialId => (
                  <div className={css.cell} key={trialId}>
                    <HumanReadableNumber num={metrics[trialId][metric.name]} />
                  </div>
                ))}
              </div>
            ))
          }
          <div className={[ css.row, css.header, css.spanAll ].join(' ')}>
            <div className={[ css.cell, css.header, css.spanAll ].join(' ')}>
              Hyperparameters
              <SelectFilter
                disableTags
                dropdownMatchSelectWidth={200}
                label=""
                mode="multiple"
                showArrow
                value={selectedHyperparameters}
                onChange={onHyperparameterSelect}>
                {hyperparameterNames.map(hp => <Option key={hp} value={hp}>{hp}</Option>)}
              </SelectFilter>
            </div>
          </div>
          {selectedHyperparameters.map(hp => (
            <div className={css.row} key={hp}>
              <div className={[ css.cell, css.header, css.sticky, css.indent ].join(' ')}>{hp}</div>
              {trials.map(trialId => {
                const value = trialsDetails[trialId].hyperparameters[hp];
                const stringValue = JSON.stringify(value);
                return (
                  <div className={css.cell} key={trialId}>
                    {isNumber(value) ? <HumanReadableNumber num={value} /> : (
                      <Tooltip title={stringValue}>{stringValue}</Tooltip>
                    )}
                  </div>
                );
              })}
            </div>
          ))}
        </>
      ) : <Spinner spinning={!isLoaded} />}
    </div>
  );
};

export default TrialsComparisonModal;
