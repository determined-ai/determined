import { Button, Modal } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback } from 'react';

import { cancelExperiment, killExperiment } from 'services/api';
import { ExperimentBase } from 'types';

export enum ActionType {
  None,
  Cancel,
  Kill,
}

interface Props extends Omit<ModalFuncProps, 'type'> {
  experiment: ExperimentBase;
  onClose: (type: ActionType) => void;
}

const StopExperimentModal: React.FC<Props> = ({ experiment, onClose }: Props) => {

  const onOk = useCallback(async (type: ActionType) => {
    if (type === ActionType.Cancel) {
      await cancelExperiment({ experimentId: experiment.id });
    }
    if (type === ActionType.Kill) {
      await killExperiment({ experimentId: experiment.id });
    }
    onClose(type);
  }, [ experiment.id, onClose ]);

  return (
    <Modal
      footer={(<>
        <Button onClick={() => onClose(ActionType.None)}>Cancel</Button>
        <Button type="primary" onClick={() => onOk(ActionType.Kill)}>Stop Now</Button>
        <Button
          type="primary"
          onClick={() => onOk(ActionType.Cancel)}>Save Checkpoint and Stop
        </Button>
      </>)}
      title={`Stop Experiment ${experiment.id}`}
      visible={true}
      onCancel={() => onClose(ActionType.None)}
    >
      <p>Confirm stopping experiment {experiment.id}.</p>
    </Modal>
  );
};

export default StopExperimentModal;
