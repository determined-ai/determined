import { ModalFuncProps } from 'antd/es/modal/Modal';
import { useCallback } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { patchModel } from 'services/api';
import useModal, { ModalHooks as Hooks, ModalCloseReason } from 'shared/hooks/useModal/useModal';
import { ModelItem } from 'types';
import handleError from 'utils/error';

type FormInputs = {
  modelName: string;
  description?: string;
};

interface Props {
  model: ModelItem;
  onClose?: (reason?: ModalCloseReason) => void;
  fetchModel: () => Promise<void>;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: () => void;
}

const FORM_ID = 'edit-model-form';

const useModalModelEdit = ({ onClose, model, fetchModel }: Props): ModalHooks => {
  const [form] = Form.useForm<FormInputs>();
  const { modalOpen: openOrUpdate, ...modalHook } = useModal({ onClose });

  const getModalProps = useCallback((): ModalFuncProps => {
    const handleOk = async () => {
      const values = await form.validateFields();
      try {
        await patchModel({
          body: { description: values.description, name: values.modelName },
          modelName: model.name,
        });
        await fetchModel();
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to edit model',
          silent: false,
        });
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
            initialValue={model.name}
            label="Name"
            name="modelName"
            rules={[{ message: 'Name is required', required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item initialValue={model.description} label="Description" name="description">
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
  }, [fetchModel, form, model.description, model.name, onClose]);

  const modalOpen = useCallback(() => {
    openOrUpdate(getModalProps());
  }, [getModalProps, openOrUpdate]);

  return { modalOpen, ...modalHook };
};

export default useModalModelEdit;
