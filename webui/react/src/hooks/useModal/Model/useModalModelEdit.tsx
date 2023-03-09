import { ModalFuncProps } from 'antd/es/modal/Modal';
import React from 'react';
import { useCallback } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import useModal, { ModalHooks as Hooks, ModalCloseReason } from 'shared/hooks/useModal/useModal';
import handleError from 'utils/error';

type FormInputs = {
  modelName: string;
};

interface Props {
  modelName: string;
  onClose?: (reason?: ModalCloseReason) => void;
  onSaveName: (editedName: string) => Promise<Error | void>;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: () => void;
}

const FORM_ID = 'edit-model-form';

const useModalModelEdit = ({ onClose, modelName, onSaveName }: Props): ModalHooks => {
  const [form] = Form.useForm<FormInputs>();
  const { modalOpen: openOrUpdate, ...modalHook } = useModal({ onClose });

  const getModalProps = useCallback((): ModalFuncProps => {
    const handleOk = async () => {
      const values = await form.validateFields();
      try {
        onSaveName(values.modelName);
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to edit name',
          silent: false,
        });
      } finally {
        form.resetFields();
      }
    };

    const handleClose = () => {
      form.resetFields();
      onClose?.();
    };

    return {
      closable: true,
      content: (
        <Form autoComplete="off" form={form} id={FORM_ID} layout="vertical">
          <Form.Item
            initialValue={modelName}
            label="Name"
            name="modelName"
            rules={[{ message: 'Name is required', required: true }]}>
            <Input />
          </Form.Item>
        </Form>
      ),
      icon: null,
      okButtonProps: { form: FORM_ID, htmlType: 'submit', type: 'primary' },
      okText: 'Save',
      onCancel: handleClose,
      onOk: handleOk,
      title: 'Edit',
    };
  }, [form, modelName, onClose, onSaveName]);

  const modalOpen = useCallback(() => {
    openOrUpdate(getModalProps());
  }, [getModalProps, openOrUpdate]);

  return { modalOpen, ...modalHook };
};

export default useModalModelEdit;
