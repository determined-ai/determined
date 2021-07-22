import Modal from 'antd/lib/modal/Modal';
import React from 'react';

import { ExperimentBase, TrialDetails } from 'types';

interface ModalProps {
  experiment: ExperimentBase;
  trials: TrialDetails[];
  visible: boolean;
}

interface TableProps {
  trials: TrialDetails[];
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
  return <div>{trials.map(trial => trial.id)}</div>;
};

export default TrialsComparisonModal;
