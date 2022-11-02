import { Form, Input } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo } from 'react';

import { patchWorkspace } from 'services/api';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { Workspace } from 'types';
import handleError from 'utils/error';

import css from './useModalWorkspaceEdit.module.scss';

const FORM_ID = 'edit-workspace-form';

interface FormInputs {
  workspaceName: string;
}

interface Props {
  onClose?: () => void;
  workspace: Workspace;
}

const useModalWorkspaceEdit = ({ onClose, workspace }: Props): ModalHooks => {
  const [form] = Form.useForm<FormInputs>();
  const workspaceName = Form.useWatch('workspaceName', form);

  const handleClose = useCallback(() => onClose?.(), [onClose]);

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose: handleClose });

  const modalContent = useMemo(() => {
    return (
      <Form autoComplete="off" className={css.base} form={form} id={FORM_ID} layout="vertical">
        <Form.Item
          initialValue={workspace.name}
          label="Workspace Name"
          name="workspaceName"
          rules={[{ message: 'Workspace name is required', required: true }]}>
          <Input maxLength={80} />
        </Form.Item>
      </Form>
    );
  }, [form, workspace.name]);

  const handleOk = useCallback(async () => {
    const values = await form.validateFields();
    const name = values.workspaceName;

    try {
      await patchWorkspace({ id: workspace.id, name });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to edit workspace.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [form, workspace.id]);

  const getModalProps = useMemo((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { disabled: !workspaceName, form: FORM_ID, htmlType: 'submit' },
      okText: 'Save changes',
      onOk: handleOk,
      title: 'Edit Workspace',
    };
  }, [handleOk, modalContent, workspaceName]);

  const modalOpen = useCallback(
    (initialModalProps: ModalFuncProps = {}) => {
      openOrUpdate({ ...getModalProps, ...initialModalProps });
    },
    [getModalProps, openOrUpdate],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps);
  }, [getModalProps, modalRef, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalWorkspaceEdit;
