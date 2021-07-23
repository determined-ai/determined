import Modal from 'antd/lib/modal/Modal';
import axios from 'axios';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { getTrialDetails } from 'services/api';
import { ApiState } from 'services/types';
import { isAborted } from 'services/utils';
import { CheckpointState, CheckpointWorkload, TrialDetails, TrialItem } from 'types';
import { humanReadableBytes } from 'utils/string';
import { getDuration, shortEnglishHumannizer } from 'utils/time';
import { extractMetricNames, trialDurations, TrialDurations } from 'utils/trial';
import { checkpointSize } from 'utils/types';

import css from './TrialsComparisonModal.module.scss';

interface ModalProps {
  onCancel: () => void;
  trials: TrialItem[];
  visible: boolean;
}

interface TableProps {
  trials: TrialItem[];
}

const TrialsComparisonModal: React.FC<ModalProps> =
({ onCancel, trials, visible }: ModalProps) => {
  return (
    <Modal
      title={`Experiment ${trials.first()?.experimentId} Trial Comparison`}
      visible={visible}
      onCancel={onCancel}>
      <TrialsComparisonTable trials={trials} />
    </Modal>
  );
};

const TrialsComparisonTable: React.FC<TableProps> = ({ trials }: TableProps) => {
  const [ trialsDetails, setTrialsDetails ] = useState<Record<string, ApiState<TrialDetails>>>({});
  const [ source ] = useState(axios.CancelToken.source());
  const [ canceler ] = useState(new AbortController());

  const fetchTrialDetails = useCallback(async (trialId) => {
    try {
      const response = await getTrialDetails({ id: trialId }, { signal: canceler.signal });
      setTrialsDetails(prev => (
        { ...prev, [trialId]: { ...prev[trialId], data: response, isLoading: false } }
      ));
    } catch (e) {
      if (!trialsDetails[trialId].error && !isAborted(e)) {
        setTrialsDetails(prev => ({ ...prev, [trialId]: { ...prev[trialId], error: e } }));
      }
    }
  }, [ canceler.signal, trialsDetails ]);

  useEffect(() => {
    return () => {
      source.cancel();
      canceler.abort();
    };
  }, [ canceler, source ]);

  useEffect(() => {
    const trialIds = trials.map(trial => {
      setTrialsDetails(prev =>
        ({
          ...prev,
          [trial.id]: {
            data: undefined,
            error: undefined,
            isLoading: true,
            source,
          },
        }));
      return trial.id;
    });
    trialIds.forEach(trialId => fetchTrialDetails(trialId));
  }, [ fetchTrialDetails, source, trials ]);

  const durations: Record<string, TrialDurations> = useMemo(
    () => Object.fromEntries(Object.values(trialsDetails)
      .map(trial => (trial.data ? [ trial.data.id, trialDurations(trial.data.workloads) ] : [])))
    , [ trialsDetails ],
  );

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
      .map(trial => trial.data ? [ trial.data.id, getCheckpointSize(trial.data) ] : []))
    , [ getCheckpointSize, trialsDetails ],
  );

  const metricNames = useMemo(() => extractMetricNames(
    Object.values(trialsDetails).first()?.data?.workloads || [],
  ), [ trialsDetails ]);

  const hyperparameterNames = useMemo(
    () =>
      Object.keys(trials.first().hyperparameters),
    [ trials ],
  );

  return (
    <div className={css.tableContainer}>
      <div className={css.headerRow}><div />{trials.map(trial => trial.id)}</div>
      <div className={css.row}><h3>State</h3>{trials.map(trial => trial.state)}</div>
      <div className={css.row}>
        <h3>Start Time</h3>
        {trials.map(trial =>
          shortEnglishHumannizer(getDuration({ startTime: trial.startTime })) + ' ago')}
      </div>
      <div className={css.row}>
        <h3>Training Time</h3>
        {trials.map(trial => shortEnglishHumannizer(durations[trial.id]?.train))}
      </div>
      <div className={css.row}>
        <h3>Validation Time</h3>
        {trials.map(trial => shortEnglishHumannizer(durations[trial.id]?.validation))}
      </div>
      <div className={css.row}>
        <h3>Checkpoint Time</h3>
        {trials.map(trial => shortEnglishHumannizer(durations[trial.id]?.checkpoint))}
      </div>
      <div className={css.row}>
        <h3>Batches Processed</h3>
        {trials.map(trial => trial.totalBatchesProcessed)}
      </div>
      <div className={css.row}>
        <h3>Best Checkpoint</h3>
        {trials.map(trial => trial.bestAvailableCheckpoint?.totalBatches)}
      </div>
      <div className={css.row}>
        <h3>Total Checkpoint Size</h3>
        {trials.map(trial => totalCheckpointsSizes[trial.id])}
      </div>
      <div className={css.headerRow}><h2>Metrics</h2></div>
      {metricNames.map(metric =>
        <div className={css.row} key={metric.name}>
          <h3>{metric.name}</h3>
          {trials.map(trial => trialsDetails[trial.id].data?.workloads
            .find(workload =>
              Object.keys(workload.training?.metrics || {}).first() === metric.name ||
              Object.keys(workload.validation?.metrics || {}).first() === metric.name))}
        </div>)}
      <div className={css.headerRow}><h2>Hyperparameters</h2></div>
      {hyperparameterNames.map(hp =>
        trials.map(trial => trial.hyperparameters[hp]))}
    </div>
  );
};

export default TrialsComparisonModal;
