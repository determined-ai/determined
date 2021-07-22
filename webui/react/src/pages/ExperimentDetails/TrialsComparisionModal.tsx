import Modal from 'antd/lib/modal/Modal';
import React from 'react';

import { ExperimentBase, TrialDetails } from 'types';

interface ModalProps {
  experiment: ExperimentBase;
  trials: TrialDetails[];
}

interface TableProps {
  trials: TrialDetails[];
}

const TrialsComparisonModal: React.FC<ModalProps> = ({ experiment, trials }: ModalProps) => {
  return (
    <Modal title={`Experiment ${experiment.id} Trial Comparison`}>
      <TrialsComparisonTable trials={trials} />
    </Modal>
  );
};

const TrialsComparisonTable: React.FC<TableProps> = ({ trials }: TableProps) => {
  return <div>{trials.map(trial => trial.id)}</div>;
};

export default TrialsComparisonModal;
