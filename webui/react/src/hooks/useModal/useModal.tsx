/* eslint-disable @typescript-eslint/no-explicit-any */
import { Modal } from 'antd';
import { ModalFunc } from 'antd/es/modal/confirm';
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
  style: { minWidth: 600 },
  visible: true,
};

type AntModalPromise = (...args: any[]) => any;

const useModal = (
  onClose?: (reason: ModalCloseReason) => void,
  options?: ModalOptions,
): ModalHooks => {
  const modalRef = useRef<ReturnType<ModalFunc>>();
  const [ modalProps, setModalProps ] = useState<ModalFuncProps>();
  const prevModalProps = usePrevious(modalProps, undefined);

  const modalOpen = useCallback((props: ModalFuncProps = {}) => {
    setModalProps(props);
  }, []);

  const modalClose = useCallback((reason?: ModalCloseReason) => {
    if (!modalRef.current) return;
    modalRef.current.destroy();
    modalRef.current = undefined;
    if (reason) onClose?.(reason);
  }, [ onClose ]);

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
          .catch(reject);
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
      onCancel: options?.rawCancel
        ? modalProps.onCancel
        : extendEventHandler(modalProps.onCancel, ModalCloseReason.Cancel),
      onOk: options?.rawOk
        ? modalProps.onOk
        : extendEventHandler(modalProps.onOk, ModalCloseReason.Ok),
    };

    // Update the modal if it already exists, otherwise open a new modal.
    if (modalRef.current) {
      modalRef.current.update(completeModalProps);
    } else {
      modalRef.current = Modal.confirm(completeModalProps);
    }
  }, [ extendEventHandler, modalProps, options, prevModalProps ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModal;
