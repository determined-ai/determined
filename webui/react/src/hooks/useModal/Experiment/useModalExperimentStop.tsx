import { Alert } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import { CheckboxChangeEvent } from 'antd/lib/checkbox';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Checkbox from 'components/kit/Checkbox';
import { cancelExperiment, killExperiment } from 'services/api';
import useModal, { ModalCloseReason, ModalHooks } from 'shared/hooks/useModal/useModal';
import { ValueOf } from 'shared/types';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

import css from './useModalExperimentStop.module.scss';

export const ActionType = {
  Cancel: 'Cancel',
  Kill: 'Kill',
} as const;

export type ActionType = ValueOf<typeof ActionType>;

interface Props {
  experimentId: number;
  onClose?: (type?: ActionType) => void;
}

const useModalExperimentStop = ({ experimentId, onClose }: Props): ModalHooks => {
  const [type, setType] = useState<ActionType>(ActionType.Cancel);

  const handleClose = useCallback(
    (reason?: ModalCloseReason) => {
      onClose?.(reason === ModalCloseReason.Ok ? type : undefined);
    },
    [onClose, type],
  );

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose: handleClose });

  const modalContent = useMemo(() => {
    const isCancel = type === ActionType.Cancel;
    const handleCheckBoxChange = (event: CheckboxChangeEvent) => {
      setType(event.target.checked ? ActionType.Cancel : ActionType.Kill);
    };
    return (
      <div className={css.base}>
        <div>Are you sure you want to stop experiment {experimentId}?</div>
        <Checkbox checked={isCancel} onChange={handleCheckBoxChange}>
          Save checkpoint before stopping
        </Checkbox>
        {!isCancel && (
          <Alert
            className={css.error}
            message={'Note: Any progress/data on incomplete workflows will be lost.'}
            type="warning"
          />
        )}
      </div>
    );
  }, [experimentId, type]);

  const handleOk = useCallback(async () => {
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
        publicSubject: 'Unable to stop experiment.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [experimentId, type]);

  const modalProps: ModalFuncProps = useMemo(() => {
    return {
      content: modalContent,
      okText: 'Stop Experiment',
      onOk: handleOk,
      title: 'Confirm Stop',
    };
  }, [handleOk, modalContent]);

  const modalOpen = useCallback(
    (initialModalProps: ModalFuncProps = {}) => {
      openOrUpdate({ ...modalProps, ...initialModalProps });
    },
    [modalProps, openOrUpdate],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(modalProps);
  }, [modalProps, modalRef, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalExperimentStop;
