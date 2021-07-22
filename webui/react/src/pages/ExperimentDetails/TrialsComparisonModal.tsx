import Modal from 'antd/lib/modal/Modal';
import axios from 'axios';
import React, { useCallback, useEffect, useState } from 'react';

import { getTrialDetails } from 'services/api';
import { ApiState } from 'services/types';
import { isAborted } from 'services/utils';
import { TrialDetails, TrialItem } from 'types';

import css from './TrialsComparisonModal.module.scss';

interface ModalProps {
  trials: TrialItem[];
  visible: boolean;
}

interface TableProps {
  trials: TrialItem[];
}

const TrialsComparisonModal: React.FC<ModalProps> =
({ trials, visible }: ModalProps) => {
  return (
    <Modal title={`Experiment ${trials.first().experimentId} Trial Comparison`} visible={visible}>
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

  return (
    <div className={css.tableContainer}>
      <div className={css.headerRow}><div />{trials.map(trial => trial.id)}</div>
      <div className={css.row}><h3>State</h3>{trials.map(trial => trial.state)}</div>
      <div className={css.row}><h3>Start Time</h3>{trials.map(trial => trial.startTime)}</div>
      <div className={css.row}>
        <h3>Training Time</h3>
        {trials.map(trial => 'Need Workloads' + trial.id)}
      </div>
      <div className={css.row}>
        <h3>Validation Time</h3>
        {trials.map(trial => 'Need Workloads' + trial.id)}
      </div>
      <div className={css.row}>
        <h3>Checkpoint Time</h3>
        {trials.map(trial => 'Need Workloads' + trial.id)}
      </div>
      <div className={css.row}>
        <h3>Batches Processed</h3>
        {trials.map(trial => trial.totalBatchesProcessed)}
      </div>
      <div className={css.row}>
        <h3>Best Checkpoint</h3>
        {trials.map(trial => trial.bestAvailableCheckpoint)}
      </div>
      <div className={css.row}>
        <h3>Total Checkpoint Size</h3>
        {trials.map(trial => 'Need Workloads'+ trial.id)}
      </div>
      <div className={css.headerRow}><h2>Metrics</h2></div>
      {trials.map(trial => 'Need Workloads'+ trial.id)}
      <div className={css.headerRow}><h2>Hyperparameters</h2></div>
      {Object.keys(trials.first().hyperparameters).map(hp =>
        trials.map(trial => trial.hyperparameters[hp]))}
    </div>
  );
};

export default TrialsComparisonModal;
