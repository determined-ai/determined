/* eslint-disable @typescript-eslint/no-explicit-any */
import { Modal } from 'antd';
import { ModalFunc } from 'antd/es/modal/confirm';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useRef } from 'react';

export interface ModalHooks {
  modalClose: () => void;
  modalOpen: (modalProps?: ModalFuncProps) => void;
  modalRef: React.MutableRefObject<ReturnType<ModalFunc> | undefined>;
}

const DEFAULT_MODAL_PROPS: Partial<ModalFuncProps> = { style: { minWidth: 600 } };

type AntModalPromise = (...args: any[]) => any;

/*
 * This utility function is needed for `antd` modal `onOk` handlers.
 * If an async function is passed directly to `onOk` the modal
 * will NOT block the UI with a spinner on the `Ok` button.
 * Wrapping a promise around the async function is the current work
 * around until `antd` supports async handlers in the future.
 */
export const asyncToPromise = (fn: any): AntModalPromise => {
  return (...args: any[]) => new Promise((resolve, reject) => {
    return fn(...args).then(resolve).catch(reject);
  }) as unknown as AntModalPromise;
};

const useModal = (onClose?: () => void): ModalHooks => {
  const modalRef = useRef<ReturnType<ModalFunc>>();

  const modalOpen = useCallback((props: ModalFuncProps = {}) => {
    const modalProps = { ...DEFAULT_MODAL_PROPS, ...props };
    if (modalRef.current) {
      modalRef.current.update(prev => ({ ...prev, ...modalProps }));
    } else {
      modalRef.current = Modal.confirm(modalProps);
    }
  }, []);

  const modalClose = useCallback(() => {
    if (!modalRef.current) return;
    modalRef.current.destroy();
    modalRef.current = undefined;
    onClose?.();
  }, [ onClose ]);

  // When the component using the hook unmounts, remove the modal automatically.
  useEffect(() => {
    return () => modalClose();
  }, [ modalClose ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModal;
