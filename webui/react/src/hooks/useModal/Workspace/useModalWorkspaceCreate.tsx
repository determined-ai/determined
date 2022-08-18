import { Form, Input } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo } from 'react';

import { paths } from 'routes/utils';
import { createWorkspace } from 'services/api';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { DetError, ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import handleError from 'utils/error';

interface FormInputs {
  workspaceName: string;
}

interface Props {
  onClose?: () => void;
}

const useModalWorkspaceCreate = ({ onClose }: Props = {}): ModalHooks => {
  const [ form ] = Form.useForm<FormInputs>();

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });

  const modalContent = useMemo(() => {
    return (
      <Form autoComplete="off" form={form} layout="vertical">
        <Form.Item
          label="Workspace Name"
          name="workspaceName"
          rules={[ { message: 'Workspace name is required ', required: true } ]}>
          <Input maxLength={80} />
        </Form.Item>
      </Form>
    );
  }, [ form ]);

  const handleOk = useCallback(async () => {
    const values = await form.validateFields();

    try {
      if (values) {
        const response = await createWorkspace({ name: values.workspaceName });
        routeToReactUrl(paths.workspaceDetails(response.id));
      }
    } catch (e) {
      if (e instanceof DetError) {
        handleError(e, {
          level: e.level,
          publicMessage: e.publicMessage,
          publicSubject: 'Unable to create workspace.',
          silent: false,
          type: e.type,
        });
      } else {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to create workspace.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    }
  }, [ form ]);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okText: 'Create Workspace',
      onOk: handleOk,
      title: 'New Workspace',
    };
  }, [ handleOk, modalContent ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    openOrUpdate({ ...getModalProps(), ...initialModalProps });
  }, [ getModalProps, openOrUpdate ]);

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps());
  }, [ getModalProps, modalRef, openOrUpdate ]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalWorkspaceCreate;
