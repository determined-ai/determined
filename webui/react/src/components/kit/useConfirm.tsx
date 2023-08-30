import React, { PropsWithChildren, ReactNode, useState } from 'react';

import { ErrorHandler } from 'components/kit/internal/types';
import { Modal, ModalSize, useModal } from 'components/kit/Modal';

export interface ConfirmModalProps {
  cancelText?: string;
  content?: ReactNode;
  danger?: boolean;
  size?: ModalSize;
  title?: string;
  okText?: string;
  onClose?: () => void;
  onConfirm: () => Promise<void> | void;
  onError: ErrorHandler;
}

export const DEFAULT_CONFIRM_TITLE = 'Confirm Action';
export const DEFAULT_CONFIRM_LABEL = 'Confirm';
export const DEFAULT_CONTENT = 'Are you sure?';

const ConfirmModal = ({
  cancelText,
  content,
  danger = false,
  size = 'small',
  title,
  okText,
  onClose,
  onConfirm,
  onError,
}: ConfirmModalProps) => {
  return (
    <Modal
      cancel
      cancelText={cancelText}
      danger={danger}
      icon="warning-large"
      size={size}
      submit={{
        handleError: onError,
        handler: onConfirm,
        text: okText ?? DEFAULT_CONFIRM_LABEL,
      }}
      title={title ?? DEFAULT_CONFIRM_TITLE}
      onClose={onClose}>
      <div>{content}</div>
    </Modal>
  );
};

type ConfirmModalModifier = (args: ConfirmModalProps) => void;

/* eslint-disable @typescript-eslint/no-empty-function */
export const voidFn = (): void => {};
export const voidPromiseFn = async (): Promise<void> => {};
const ConfirmationContext = React.createContext<ConfirmModalModifier | null>(null);

export const ConfirmationProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [modalProps, setModalProps] = useState<ConfirmModalProps>();
  const Modal = useModal(ConfirmModal);

  const contextValue = ({
    cancelText,
    content = DEFAULT_CONTENT,
    danger = false,
    okText,
    size = 'small',
    title,
    onClose = voidFn,
    onConfirm = voidPromiseFn,
    onError = voidFn,
  }: ConfirmModalProps) => {
    setModalProps({
      cancelText,
      content,
      danger,
      okText,
      onClose,
      onConfirm,
      onError,
      size,
      title,
    });
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
        {...modalProps}
        onConfirm={modalProps?.onConfirm ?? voidPromiseFn}
        onError={modalProps?.onError ?? voidFn}
      />
    </>
  );
};

const useConfirm = (): ConfirmModalModifier => {
  const context = React.useContext(ConfirmationContext);
  if (context === null) {
    throw new Error('Attempted to use confirmation modal outside of ConfirmationContext');
  }
  return context;
};

export default useConfirm;
