import { Alert } from 'antd';
import React from 'react';

import { Modal } from 'components/kit/Modal';

import { killTask } from '../services/api';
import { ErrorLevel, ErrorType } from '../shared/utils/error';
import { CommandTask } from '../types';
import handleError from '../utils/error';

export const BUTTON_TEXT = 'Confirm Task Kill';

interface Props {
  task: CommandTask;
  onClose?: () => void;
  onKill?: () => void;
}
const TaskKillModalComponent: React.FC<Props> = ({ task, onClose, onKill }: Props) => {
  const handleSubmit = async () => {
    try {
      await killTask(task);
      onKill?.();
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to stop task.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  };

  return (
    <Modal
      size="small"
      submit={{
        handler: handleSubmit,
        text: BUTTON_TEXT,
      }}
      title="Confirm Kill"
      onClose={onClose}>
      <div>Are you sure you want to stop task <code>{task.id}</code>?</div>
      <Alert
        message={'Note: Any progress/data on incomplete workflows will be lost.'}
        type="warning"
      />
    </Modal>
  );
};

export default TaskKillModalComponent;
