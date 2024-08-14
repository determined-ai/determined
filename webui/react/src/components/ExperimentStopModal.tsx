import Alert from 'hew/Alert';
import Checkbox, { CheckboxChangeEvent } from 'hew/Checkbox';
import { Modal } from 'hew/Modal';
import React, { useState } from 'react';

import useFeature from 'hooks/useFeature';
import { cancelExperiment, killExperiment } from 'services/api';
import { ExperimentAction, ValueOf } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { capitalize } from 'utils/string';

export const ActionType = {
  Cancel: ExperimentAction.Cancel,
  Kill: ExperimentAction.Kill,
} as const;

export type AvalableActions = ValueOf<typeof ActionType>;

export const CHECKBOX_TEXT = 'Save checkpoint before stopping';

interface Props {
  experimentId: number;
  onClose?: () => void;
}

const ExperimentStopModalComponent: React.FC<Props> = ({ experimentId, onClose }: Props) => {
  const [type, setType] = useState<AvalableActions>(ActionType.Cancel);
  const f_flat_runs = useFeature().isOn('flat_runs');

  const actionCopy = f_flat_runs ? 'stop search' : 'stop experiment';

  const handleCheckBoxChange = (event: CheckboxChangeEvent) => {
    setType(event.target.checked ? ActionType.Cancel : ActionType.Kill);
  };

  const handleSubmit = async () => {
    try {
      if (type === ActionType.Cancel) {
        await cancelExperiment({ experimentId });
      } else {
        await killExperiment({ experimentId });
      }
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: `Unable to ${actionCopy}.`,
        silent: false,
        type: ErrorType.Server,
      });
    }
  };

  return (
    <Modal
      size="small"
      submit={{
        handleError,
        handler: handleSubmit,
        text: capitalize(actionCopy),
      }}
      title="Confirm Stop"
      onClose={onClose}>
      <div>
        Are you sure you want to {actionCopy} {experimentId}?
      </div>
      <Checkbox checked={type === ActionType.Cancel} onChange={handleCheckBoxChange}>
        {CHECKBOX_TEXT}
      </Checkbox>
      {type !== ActionType.Cancel && (
        <Alert
          message="Note: Any progress/data on incomplete workflows will be lost."
          type="warning"
        />
      )}
    </Modal>
  );
};

export default ExperimentStopModalComponent;
