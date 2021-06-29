import { Select, Tag } from 'antd';
import Modal from 'antd/lib/modal/Modal';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import HumanReadableFloat from 'components/HumanReadableFloat';
import Link from 'components/Link';
import MetricBadgeTag from 'components/MetricBadgeTag';
import MetricSelectFilter from 'components/MetricSelectFilter';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import useResize from 'hooks/useResize';
import { paths } from 'routes/utils';
import { getTrialDetails } from 'services/api';
import { CheckpointWorkload, ExperimentBase,
  MetricName, MetricsWorkload, TrialDetails } from 'types';
import { humanReadableBytes } from 'utils/string';
import { shortEnglishHumannizer } from 'utils/time';
import { extractMetricNames, trialDurations, TrialDurations } from 'utils/trial';
import { checkpointSize } from 'utils/types';

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
      style={{ height: resize.height*.9 }}
      title={`Experiment ${experiment.id} Trial Comparison`}
      visible={visible}
      width={resize.width*.9}
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
    } catch {
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

  const durations: Record<string, TrialDurations> = useMemo(
    () => Object.fromEntries(Object.values(trialsDetails)
      .map(trial => [ trial.id, trialDurations(trial.workloads) ]))
    , [ trialsDetails ],
  );

  const getCheckpointSize = useCallback((trial: TrialDetails) => {
    const totalBytes = trial.workloads
      .filter(step => !!step.checkpoint)
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
          <div className={css.headerRow}>
            <div />
            {trials.map(trial => (
              <Tag
                className={[ css.trialTag, css.centerVertically ].join(' ')}
                closable
                key={trial}
                onClose={() => handleTrialUnselect(trial)}>
                <Link path={paths.trialDetails(trial, experiment.id)}>Trial {trial}</Link>
              </Tag>
            ))}
          </div>
          <div className={css.tableBody}>
            <div className={css.row}>
              <h3>State</h3>
              {trials.map(trial => (
                <div className={css.centerVertically} key={trial}>
                  <Badge state={trialsDetails[trial].state} type={BadgeType.State} />
                </div>
              ))}
            </div>
            <div className={css.row}>
              <h3>Training Time</h3>
              {trials.map(trial => (
                <div key={trial}>{shortEnglishHumannizer(durations[trial]?.train)}</div>
              ))}
            </div>
            <div className={css.row}>
              <h3>Batches Processed</h3>
              {trials.map(trial =>
                <div key={trial}>{trialsDetails[trial].totalBatchesProcessed}</div>)}
            </div>
            <div className={css.row}>
              <h3>Total Checkpoint Size</h3>
              {trials.map(trial => <div key={trial}>{totalCheckpointsSizes[trial]}</div>)}
            </div>
            <div className={css.headerRow}>
              <h2>Metrics</h2>
              <MetricSelectFilter
                defaultMetricNames={metricNames}
                label=""
                metricNames={metricNames}
                multiple
                value={selectedMetrics}
                onChange={onMetricSelect}
              />
            </div>
            {metricNames.filter(metric => selectedMetrics.map(m => m.name)
              .includes(metric.name)).map(metric => (
              <div className={css.row} key={metric.name}>
                <h3><MetricBadgeTag metric={metric} /></h3>
                {trials.map(trial => metrics[trial][metric.name] ? (
                  <div><HumanReadableFloat key={trial} num={metrics[trial][metric.name]} /></div>
                ) : <div />)}
              </div>
            ))}
            <div className={css.headerRow}><h2>Hyperparameters</h2>
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
            {selectedHyperparameters.map(hp => (
              <div className={css.row} key={hp}>
                <h3>{hp}</h3>
                {trials.map(trial => (
                  !isNaN(parseFloat(JSON.stringify(trialsDetails[trial].hyperparameters[hp]))) ? (
                    <div>
                      <HumanReadableFloat
                        key={trial}
                        num={parseFloat(JSON.stringify(trialsDetails[trial].hyperparameters[hp]))}
                      />
                    </div>
                  ) : (
                    <div>{String(trialsDetails[trial].hyperparameters[hp])}</div>
                  )
                ))}
              </div>
            ))}
          </div>
        </>
      ) : <Spinner spinning={!isLoaded} />}
    </div>
  );
};

export default TrialsComparisonModal;
