import React from 'react';

import { Modal } from 'components/kit/Modal';
import { UNMANAGED_EXPERIMENT_ANNOTATION_MESSAGE } from 'constant';
import { ExperimentAction } from 'types';
import handleError from 'utils/error';

interface Props {
  batchAction: ExperimentAction;
  itemName?: string;
  onConfirm: () => Promise<void>;
  onClose?: () => void;
}

export const CONFIRM_BUTTON_LABEL = 'Confirm';

const DANGEROUS_BATCH_ACTIONS: ExperimentAction[] = [
  ExperimentAction.Cancel,
  ExperimentAction.Delete,
  ExperimentAction.Kill,
];

const BatchActionConfirmModalComponent: React.FC<Props> = ({
  batchAction,
  itemName = 'experiment',
  onConfirm,
  onClose,
}: Props) => {
  const danger = DANGEROUS_BATCH_ACTIONS.includes(batchAction);

  return (
    <Modal
      cancel
      danger={danger}
      icon="info"
      size="small"
      submit={{
        handleError,
        handler: onConfirm,
        text: batchAction === ExperimentAction.Cancel ? CONFIRM_BUTTON_LABEL : batchAction,
      }}
      title={`Confirm Batch ${batchAction}`}
      onClose={onClose}>
      <div>
        Are you sure you want to <b>{batchAction.toLocaleLowerCase()}</b> all selected {itemName}s?
      </div>
      <div>
        <small>{UNMANAGED_EXPERIMENT_ANNOTATION_MESSAGE}</small>
      </div>
    </Modal>
  );
};

export default BatchActionConfirmModalComponent;
