/* eslint-disable @typescript-eslint/no-explicit-any */
import { Modal } from 'antd';
import { ModalFunc, ModalStaticFunctions } from 'antd/es/modal/confirm';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import { isAsyncFunction } from 'utils/data';

import usePrevious from '../usePrevious';

export interface ModalHooks {
  modalClose: (reason?: ModalCloseReason) => void;
  modalOpen: (modalProps?: ModalFuncProps) => void;
  modalRef: React.MutableRefObject<ReturnType<ModalFunc> | undefined>;
}

/*
 * By default we add modal close calls to the tail end of both `Ok` and
 * `Cancel` callbacks. `rawCancel` and `rawOk` allow us to override and
 * skip this automatic addition as we might not want this behavior.
 * For example, there may be different modes within the modal and the
 * cancel button might be used to toggle between the modes instead of
 * cancelling out of the modal.
 */
interface ModalOptions {
  rawCancel?: boolean;
  rawOk?: boolean;
}

export enum ModalCloseReason {
  Cancel = 'Cancel',
  Ok = 'Ok',
}

const DEFAULT_MODAL_PROPS: Partial<ModalFuncProps> = {
  maskClosable: true,
  style: { minWidth: 280 },
  visible: true,
};

type AntModalPromise = (...args: any[]) => any;

const useModal = (config: {
  modal?: Omit<ModalStaticFunctions, 'warn'>,
  onClose?: (reason: ModalCloseReason) => void,
  options?: ModalOptions,
} = {}): ModalHooks => {
  const modalRef = useRef<ReturnType<ModalFunc>>();
  const componentUnmounting = useRef(false);
  const [ modalProps, setModalProps ] = useState<ModalFuncProps>();
  const prevModalProps = usePrevious(modalProps, undefined);

  const modalOpen = useCallback((props: ModalFuncProps = {}) => {
    setModalProps(props);
  }, []);

  const modalClose = useCallback((reason?: ModalCloseReason) => {
    if (!modalRef.current) return;
    modalRef.current.destroy();
    modalRef.current = undefined;
    if (reason) config.onClose?.(reason);
  }, [ config ]);

  /*
   * Adds `modalClose` to event handlers `onOk` and `onCancel`.
   * Handles `undefined`, asynchronous and synchronous event handlers.
   */
  const extendEventHandler = useCallback((fn: any, reason: ModalCloseReason): AntModalPromise => {
    if (fn === undefined) {
      return () => modalClose(reason);
    } else if (isAsyncFunction(fn)) {
      return (...args: any[]) => new Promise((resolve, reject) => {
        return fn(...args)
          .then((...thenArgs: any[]) => {
            resolve(thenArgs);
            modalClose(reason);
          })
          .catch((e: unknown) => reject(e));
      }) as unknown as AntModalPromise;
    } else {
      return async (...args: any[]) => {
        await fn(...args);
        modalClose(reason);
      };
    }
  }, [ modalClose ]);

  useEffect(() => {
    // Only render/re-render when modal props have changed.
    if (!modalProps || modalProps === prevModalProps) return;

    const completeModalProps: ModalFuncProps = {
      ...DEFAULT_MODAL_PROPS,
      ...modalProps,
      onCancel: config.options?.rawCancel
        ? modalProps.onCancel
        : extendEventHandler(modalProps.onCancel, ModalCloseReason.Cancel),
      onOk: config.options?.rawOk
        ? modalProps.onOk
        : extendEventHandler(modalProps.onOk, ModalCloseReason.Ok),
    };

    // Update the modal if it already exists, otherwise open a new modal.
    if (modalRef.current) {
      modalRef.current.update(completeModalProps);
    } else {
      modalRef.current = (config.modal || Modal).confirm(completeModalProps);
    }
  }, [ config, extendEventHandler, modalProps, prevModalProps ]);

  /**
   * Sets componentUnmounting to true only when the parent component is unmounting so that the next
   * useEffect only runs modalClose on unmount, rather than every time modalClose updates.
   * The order of these two useEffects matters, this one has to be first.
   */
  useEffect(() => {
    return () => {
      componentUnmounting.current = true;
    };
  }, []);

  // When the component using the hook unmounts, remove the modal automatically.
  useEffect(() => {
    return () => {
      if (componentUnmounting.current) modalClose();
    };
  }, [ modalClose ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModal;
