import React, { ReactNode } from 'react';

import { Modal } from './kit/Modal';

interface Props {
  content: ReactNode;
  danger?: boolean;
  onClose?: () => void;
  onOk?: () => Promise<void>;
  okText?: string;
  title: string;
}
const ConfirmModalComponent: React.FC<Props> = (
  {
    content,
    danger,
    okText,
    onClose,
    onOk,
    title,
  }: Props,
) => {
  const submit = onOk ? {
    handler: onOk,
    text: okText ?? 'Confirm',
  } : undefined;

  return (
    <Modal
      cancel
      danger={danger}
      icon="info"
      size="small"
      submit={submit}
      title={title}
      onClose={onClose}>
      <div>{content}</div>
    </Modal>
  );
};

export default ConfirmModalComponent;
