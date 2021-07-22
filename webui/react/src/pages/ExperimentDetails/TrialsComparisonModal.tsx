import Modal from 'antd/lib/modal/Modal';
import React from 'react';

import { ExperimentBase, TrialItem } from 'types';

import css from './TrialsComparisonModal.module.scss';

interface ModalProps {
  experiment: ExperimentBase;
  trials: TrialItem[];
  visible: boolean;
}

interface TableProps {
  trials: TrialItem[];
}

const TrialsComparisonModal: React.FC<ModalProps> =
({ experiment, trials, visible }: ModalProps) => {
  return (
    <Modal title={`Experiment ${experiment.id} Trial Comparison`} visible={visible}>
      <TrialsComparisonTable trials={trials} />
    </Modal>
  );
};

const TrialsComparisonTable: React.FC<TableProps> = ({ trials }: TableProps) => {
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
