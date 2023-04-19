import React, { PropsWithChildren, ReactNode, useState } from 'react';

import { Modal, useModal } from './Modal';

export interface ConfirmModalProps {
  cancelText?: string;
  content?: ReactNode;
  danger?: boolean;
  title?: string;
  okText?: string;
  onClose?: () => void;
  onConfirm?: () => Promise<void>;
}

export const DEFAULT_CONFIRM_TITLE = 'Confirm Action';
export const DEFAULT_CONFIRM_LABEL = 'Confirm';
export const DEFAULT_CONTENT = 'Are you sure?';

const ConfirmModal = ({
  cancelText,
  content,
  danger = false,
  title,
  okText,
  onClose,
  onConfirm,
}: ConfirmModalProps) => {
  return (
    <Modal
      cancel
      cancelText={cancelText}
      danger={danger}
      icon="warning-large"
      size="small"
      submit={{
        handler: onConfirm ?? voidPromiseFn,
        text: okText ?? DEFAULT_CONFIRM_LABEL,
      }}
      title={title ?? DEFAULT_CONFIRM_TITLE}
      onClose={onClose}>
      <div>{content}</div>
    </Modal>
  );
};

type VoidFn = () => void;
type VoidPromiseFn = () => Promise<void>;
type ConfirmModalModifier = (args: ConfirmModalProps) => void;

/* eslint-disable @typescript-eslint/no-empty-function */
const voidFn = () => {};
const voidPromiseFn = async () => {};
const ConfirmationContext = React.createContext<ConfirmModalModifier | null>(null);

export const ConfirmationProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [cancelText, setCancelText] = useState<string>();
  const [content, setContent] = useState<ReactNode>();
  const [danger, setDanger] = useState(false);
  const [okText, setOkText] = useState<string>();
  const [title, setTitle] = useState<string>();
  const [onClose, setOnClose] = useState<VoidFn>(voidFn);
  const [onConfirm, setOnConfirm] = useState<VoidPromiseFn>(voidPromiseFn);
  const Modal = useModal(ConfirmModal);

  const contextValue = ({
    cancelText,
    content = DEFAULT_CONTENT,
    danger = false,
    okText,
    title,
    onClose = voidFn,
    onConfirm = voidPromiseFn,
  }: ConfirmModalProps) => {
    setCancelText(cancelText);
    setContent(content);
    setDanger(danger);
    setOkText(okText);
    setTitle(title);
    setOnClose(() => onClose);
    setOnConfirm(() => onConfirm);
    Modal.open();
  };

  return (
    <>
      {React.useMemo(
        () => (
          <ConfirmationContext.Provider value={contextValue}>
            {children}
          </ConfirmationContext.Provider>
        ),
        /* eslint-disable-next-line react-hooks/exhaustive-deps */
        [children],
      )}
      <Modal.Component
        cancelText={cancelText}
        content={content}
        danger={danger}
        okText={okText}
        title={title}
        onClose={onClose}
        onConfirm={onConfirm}
      />
    </>
  );
};

export const useConfirm = (): ConfirmModalModifier => {
  const context = React.useContext(ConfirmationContext);
  if (context === null) {
    throw new Error('Attempted to use confirmation modal outside of ConfirmationContext');
  }
  return context;
};
