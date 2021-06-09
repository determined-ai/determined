import { Checkbox, Modal } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useState } from 'react';

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
  const [ type, setType ] = useState<ActionType>(ActionType.Kill);

  const onOk = useCallback(async () => {
    if (type === ActionType.Cancel) {
      await cancelExperiment({ experimentId: experiment.id });
    }
    if (type === ActionType.Kill) {
      await killExperiment({ experimentId: experiment.id });
    }
    onClose(type);
  }, [ experiment.id, onClose, type ]);

  return (
    <Modal
      okText="Stop Experiment"
      title={`Stop Experiment ${experiment.id}`}
      visible={true}
      onCancel={() => onClose(ActionType.None)}
      onOk={onOk}
    >
      <p>Confirm stopping experiment {experiment.id}.</p>
      <Checkbox
        checked={type === ActionType.Cancel}
        onChange={(e) => setType(e.target.checked ? ActionType.Cancel : ActionType.Kill)}
      >Save checkpoint before stopping experiment</Checkbox>
    </Modal>
  );
};

export default StopExperimentModal;
