import { Alert, Checkbox } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import { CheckboxChangeEvent } from 'antd/lib/checkbox';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import useModal, { asyncToPromise, ModalHooks } from 'hooks/useModal';
import { cancelExperiment, killExperiment } from 'services/api';
import { ExperimentBase } from 'types';

export enum ActionType {
  Cancel = 'Cancel',
  Kill = 'Kill',
}

interface Props extends Omit<ModalFuncProps, 'type'> {
  experiment: ExperimentBase;
  onClose?: (type?: ActionType) => void;
}

const useModalExperimentStop = ({ experiment, onClose }: Props): ModalHooks => {
  const [ type, setType ] = useState<ActionType>(ActionType.Cancel);

  const modalContent = useMemo(() => {
    const isCancel = type === ActionType.Cancel;
    const handleCheckBoxChange = (event: CheckboxChangeEvent) => {
      console.log('handleCheckBoxChange', event.target.checked);
      setType(event.target.checked ? ActionType.Cancel : ActionType.Kill);
    };
    return (
      <>
        <p>Are you sure you want to stop experiment {experiment.id}?</p>
        <Checkbox
          checked={isCancel}
          onChange={handleCheckBoxChange}>
          Save checkpoint before stopping
        </Checkbox>
        {!isCancel && (
          <p>
            <Alert
            // className={css.error}
              message={'Note: Any progress/data on incomplete workflows will be lost.'}
              type="warning"
            />
          </p>
        )}
      </>
    );
  }, [ experiment.id, type ]);

  const handleCancel = useCallback(() => onClose?.(), [ onClose ]);

  const { modalClose, modalOpen: open, modalRef } = useModal(handleCancel);

  const handleOk = useCallback(async () => {
    try {
      if (type === ActionType.Cancel) {
        await cancelExperiment({ experimentId: experiment.id });
      } else {
        await killExperiment({ experimentId: experiment.id });
      }
      onClose?.(type);
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to stop experiment.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ experiment.id, onClose, type ]);

  const modalProps: ModalFuncProps = useMemo(() => {
    return {
      content: modalContent,
      okText: 'Stop Experiment',
      onCancel: handleCancel,
      onOk: asyncToPromise(handleOk),
      title: 'Confirm Stop',
      visible: true,
    };
  }, [ handleCancel, handleOk, modalContent ]);

  const modalOpen = useCallback((props: ModalFuncProps = {}) => {
    open({ ...modalProps, ...props });
  }, [ modalProps, open ]);

  useEffect(() => {
    if (modalRef.current) modalRef.current.update(modalProps);
  }, [ modalProps, modalRef ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalExperimentStop;
