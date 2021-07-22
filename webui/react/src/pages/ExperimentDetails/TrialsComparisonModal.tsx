import Modal from 'antd/lib/modal/Modal';
import React from 'react';

import { ExperimentBase, TrialItem } from 'types';

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
  return <div>{trials.map(trial => trial.id)}</div>;
};

export default TrialsComparisonModal;
