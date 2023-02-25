import { ModalFuncProps } from 'antd/es/modal/Modal';
import React from 'react';
import { useCallback } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { patchExperiment } from 'services/api';
import useModal, { ModalHooks as Hooks, ModalCloseReason } from 'shared/hooks/useModal/useModal';
import handleError from 'utils/error';

type FormInputs = {
  experimentName: string;
};

interface Props {
  experimentId: number;
  experimentName: string;
  fetchExperimentDetails: () => void;
  onClose?: (reason?: ModalCloseReason) => void;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: () => void;
}

const FORM_ID = 'edit-experiment-form';

const useModalExperimentEdit = ({
  onClose,
  experimentName,
  experimentId,
  fetchExperimentDetails,
}: Props): ModalHooks => {
  const [form] = Form.useForm<FormInputs>();
  const { modalOpen: openOrUpdate, ...modalHook } = useModal({ onClose });

  const getModalProps = useCallback((): ModalFuncProps => {
    const handleOk = async () => {
      const values = await form.validateFields();
      try {
        await patchExperiment({
          body: { name: values.experimentName },
          experimentId,
        });
        fetchExperimentDetails();
      } catch (e) {
        handleError(e, {
          publicMessage: 'Unable to update name',
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
            initialValue={experimentName}
            label="Name"
            name="experimentName"
            rules={[{ max: 128, message: 'Name must be 1 ~ 128 characters', required: true }]}>
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
  }, [experimentId, experimentName, fetchExperimentDetails, form, onClose]);

  const modalOpen = useCallback(() => {
    openOrUpdate(getModalProps());
  }, [getModalProps, openOrUpdate]);

  return { modalOpen, ...modalHook };
};

export default useModalExperimentEdit;
