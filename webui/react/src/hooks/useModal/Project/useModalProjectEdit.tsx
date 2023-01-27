import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { patchProject } from 'services/api';
import useModal, { ModalCloseReason, ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { Project } from 'types';
import handleError from 'utils/error';

import css from './useModalProjectEdit.module.scss';

const FORM_ID = 'edit-project-form';

interface FormInputs {
  description?: string;
  projectName: string;
}

interface Props {
  onClose?: () => void;
  project: Project;
}

const useModalProjectEdit = ({ onClose, project }: Props): ModalHooks => {
  const [form] = Form.useForm<FormInputs>();
  const projectName = Form.useWatch('projectName', form);

  const { modalClose, modalOpen: openOrUpdate, modalRef, ...modalHooks } = useModal({ onClose });

  const modalContent = useMemo(() => {
    return (
      <Form autoComplete="off" className={css.base} form={form} id={FORM_ID} layout="vertical">
        <Form.Item
          initialValue={project.name}
          label="Project Name"
          name="projectName"
          rules={[{ message: 'Project name is required', required: true }]}>
          <Input maxLength={80} />
        </Form.Item>
        <Form.Item initialValue={project.description} label="Description" name="description">
          <Input />
        </Form.Item>
      </Form>
    );
  }, [form, project.description, project.name]);

  const handleOk = useCallback(async () => {
    const values = await form.validateFields();
    const name = values.projectName;
    const description = values.description;

    try {
      await patchProject({ description, id: project.id, name });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to edit project.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [form, project.id]);

  const getModalProps = useMemo((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { disabled: !projectName, form: FORM_ID, htmlType: 'submit' },
      okText: 'Save Changes',
      onCancel: () => {
        form.resetFields();
        modalClose(ModalCloseReason.Cancel);
      },
      onOk: handleOk,
      title: 'Edit Project',
    };
  }, [handleOk, modalContent, projectName, form, modalClose]);

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

  return { modalClose, modalOpen, modalRef, ...modalHooks };
};

export default useModalProjectEdit;
