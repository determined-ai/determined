import { ModalFuncProps } from 'antd';
import { useCallback } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { patchExperiment } from 'services/api';
import useModal, { ModalHooks as Hooks, ModalCloseReason } from 'shared/hooks/useModal/useModal';
import handleError from 'utils/error';

type FormInputs = {
  description: string;
  experimentName: string;
};

interface Props {
  description: string;
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
  description,
  fetchExperimentDetails,
}: Props): ModalHooks => {
  const [form] = Form.useForm<FormInputs>();
  const { modalOpen: openOrUpdate, ...modalHook } = useModal({ onClose });

  const getModalProps = useCallback((): ModalFuncProps => {
    const handleOk = async () => {
      const values = await form.validateFields();
      try {
        await patchExperiment({
          body: { description: values.description, name: values.experimentName },
          experimentId,
        });
        fetchExperimentDetails();
      } catch (e) {
        handleError(e, {
          publicMessage: 'Unable to update name',
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
            initialValue={experimentName}
            label="Name"
            name="experimentName"
            rules={[{ max: 128, message: 'Name must be 1 ~ 128 characters', required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item initialValue={description} label="Description" name="description">
            <Input.TextArea />
          </Form.Item>
        </Form>
      ),
      icon: null,
      okButtonProps: { form: FORM_ID, htmlType: 'submit', type: 'primary' },
      okText: 'Save',
      onCancel: handleClose,
      onOk: handleOk,
      title: 'Edit Experiment',
    };
  }, [description, experimentId, experimentName, fetchExperimentDetails, form, onClose]);

  const modalOpen = useCallback(() => {
    openOrUpdate(getModalProps());
  }, [getModalProps, openOrUpdate]);

  return { modalOpen, ...modalHook };
};

export default useModalExperimentEdit;
