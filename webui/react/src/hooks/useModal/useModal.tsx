/* eslint-disable @typescript-eslint/no-explicit-any */
import { Modal } from 'antd';
import { ModalFunc } from 'antd/es/modal/confirm';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import usePrevious from 'hooks/usePrevious';
import { RecordUnknown, ValueOf } from 'types';
import { isAsyncFunction } from 'utils/data';

export const ModalCloseReason = {
  Cancel: 'Cancel',
  Ok: 'Ok',
} as const;

export type ModalCloseReason = ValueOf<typeof ModalCloseReason>;

interface ModalProps<T> extends ModalFuncProps {
  /** use to provide context only available at modal open time */
  context?: T;
}

export type ModalOpen<T = RecordUnknown> = (modalProps?: ModalProps<T>) => void;

export interface ModalHooks<T = RecordUnknown> {
  contextHolder: React.ReactElement;
  modalClose: (reason?: ModalCloseReason) => void;
  modalOpen: ModalOpen<T>;
  modalRef: React.MutableRefObject<ReturnType<ModalFunc> | undefined>;
}

/**
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

type AntModalPromise = (...args: any[]) => any;

interface ModalConfig {
  onClose?: (reason?: ModalCloseReason) => void;
  options?: ModalOptions;
}

function useModal<T = RecordUnknown>(config: ModalConfig = {}): ModalHooks<T> {
  const modalRef = useRef<ReturnType<ModalFunc>>();
  const componentUnmounting = useRef(false);
  const [modalProps, setModalProps] = useState<ModalFuncProps>();
  const prevModalProps = usePrevious(modalProps, undefined);
  const [modal, antdContextHolder] = Modal.useModal();

  /**
   * contextHolders have keys now, so elements that contain multiple modal
   * contexts throw a duplicate key error.
   */
  const contextHolder = <>{antdContextHolder}</>;

  /**
   * The code to close the antd modal is separated out from the code that
   * calls the `onClose` handler to distinguish the motivation for closing.
   * `internalClose` is directly used when dealing with closing due to unmounting.
   * `modalClose` is used when we want to call `onClose` handler if applicable.
   */
  const internalClose = useCallback(() => {
    if (!modalRef.current) return;
    modalRef.current.destroy();
    modalRef.current = undefined;
  }, []);

  const modalOpen = useCallback((props: ModalFuncProps = {}) => {
    setModalProps(props);
  }, []);

  const modalClose = useCallback(
    (reason?: ModalCloseReason) => {
      internalClose();
      if (reason) {
        /**
         * We need to unpack onClose from config in order to please lint.
         * This is because eslint doesn't know whether it's an arrow function or not.
         * If not, the function has a dependency on `this`, which means the whole config
         * object has to be in the dependency array. We're not using that behavior, so we can
         * unpack it.
         */
        const onClose = config.onClose;
        onClose?.(reason);
      }
    },
    [config.onClose, internalClose],
  );

  /**
   * Adds `modalClose` to event handlers `onOk` and `onCancel`.
   * Handles `undefined`, asynchronous and synchronous event handlers.
   */
  const extendEventHandler = useCallback(
    (fn: any, reason: ModalCloseReason): AntModalPromise => {
      if (fn === undefined) {
        return () => modalClose(reason);
      } else if (isAsyncFunction(fn)) {
        return (...args: any[]) =>
          new Promise((resolve, reject) => {
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
    },
    [modalClose],
  );

  useEffect(() => {
    // Only render/re-render when modal props have changed.
    if (!modalProps || modalProps === prevModalProps) return;

    const completeModalProps: ModalFuncProps = {
      maskClosable: false,
      open: true,
      style: { minWidth: 280 },
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
      modalRef.current = modal.confirm(completeModalProps);
    }
  }, [config, extendEventHandler, modal, modalProps, prevModalProps]);

  /**
   * Sets componentUnmounting to true only when the parent component is unmounting so that the next
   * useEffect only runs internalClose on unmount, rather than every time internalClose updates.
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
      if (componentUnmounting.current) internalClose();
    };
  }, [internalClose]);

  return { contextHolder, modalClose, modalOpen, modalRef };
}

export default useModal;
