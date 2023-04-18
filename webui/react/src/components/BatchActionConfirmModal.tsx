import React from 'react';

import { ExperimentAction } from '../types';

import { Modal } from './kit/Modal';

interface Props {
  batchAction: ExperimentAction;
  itemName?: string;
  selectAll?: boolean;
  danger?: boolean;
  onConfirm: () => Promise<void>;
  onClose?: () => void;
}

export const CONFIRM_BUTTON_LABEL = 'Confirm';

const BatchActionConfirmModalComponent: React.FC<Props> = ({
  batchAction,
  itemName = 'experiment',
  selectAll,
  onConfirm,
  onClose,
}: Props) => {
  const danger = /(cancel|kill)/i.test(batchAction);
  const submit = {
    handler: onConfirm,
    text: /cancel/i.test(batchAction) ? CONFIRM_BUTTON_LABEL : batchAction,
  };

  return (
    <Modal
      cancel
      danger={danger}
      icon="info"
      size="small"
      submit={submit}
      title={`Confirm Batch ${batchAction}`}
      onClose={onClose}>
      <div>
        Are you sure you want to <b>{batchAction.toLocaleLowerCase()}</b> all eligible{' '}
        {selectAll ? `${itemName}s matching the current filters` : `selected ${itemName}s`}?
      </div>
    </Modal>
  );
};

export default BatchActionConfirmModalComponent;
