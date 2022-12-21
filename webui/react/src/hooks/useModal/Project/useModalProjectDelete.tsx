import { Form, Input } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo } from 'react';

import { deleteProject } from 'services/api';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { Project } from 'types';
import handleError from 'utils/error';

import css from './useModalProjectDelete.module.scss';

interface FormInputs {
  projectName: string;
}

interface Props {
  onClose?: () => void;
  onDelete?: () => void;
  project: Project;
}

const useModalProjectDelete = ({ onClose, project, onDelete }: Props): ModalHooks => {
  const [form] = Form.useForm<FormInputs>();
  const projectNameValue = Form.useWatch('projectName', form);

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });

  const modalContent = useMemo(() => {
    return (
      <Form autoComplete="off" className={css.base} form={form} layout="vertical">
        <p>
          Are you sure you want to delete <strong>&quot;{project.name}&quot;</strong>?
        </p>
        <p>All experiments and notes within it will also be deleted. This cannot be undone.</p>
        <Form.Item
          label={
            <div>
              Please type <strong>{project.name}</strong> to confirm
            </div>
          }
          name="projectName"
          rules={[
            {
              message: 'Please type the project name to confirm',
              pattern: new RegExp(`^${project.name}$`),
              required: true,
            },
          ]}>
          <Input autoComplete="off" />
        </Form.Item>
      </Form>
    );
  }, [form, project.name]);

  const handleOk = useCallback(async () => {
    try {
      await deleteProject({ id: project.id });
      if (onDelete) onDelete();
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete project.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [project.id, onDelete]);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { danger: true, disabled: projectNameValue !== project.name },
      okText: 'Delete Project',
      onOk: handleOk,
      title: 'Delete Project',
    };
  }, [handleOk, modalContent, project.name, projectNameValue]);

  const modalOpen = useCallback(
    (initialModalProps: ModalFuncProps = {}) => {
      openOrUpdate({ ...getModalProps(), ...initialModalProps });
    },
    [getModalProps, openOrUpdate],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps());
  }, [getModalProps, modalRef, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalProjectDelete;
