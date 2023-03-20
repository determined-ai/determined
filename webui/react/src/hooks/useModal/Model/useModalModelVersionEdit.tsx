import { ModalFuncProps } from 'antd/es/modal/Modal';
import { useCallback } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { patchModelVersion } from 'services/api';
import useModal, { ModalHooks as Hooks, ModalCloseReason } from 'shared/hooks/useModal/useModal';
import { ModelVersion } from 'types';
import handleError from 'utils/error';

type FormInputs = {
  modelVersionName: string;
  description?: string;
};

interface Props {
  modelVersion: ModelVersion;
  onClose?: (reason?: ModalCloseReason) => void;
  fetchModelVersion: () => Promise<void>;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: () => void;
}

const FORM_ID = 'edit-model-version-form';

const useModalModelVersionEdit = ({
  onClose,
  modelVersion,
  fetchModelVersion,
}: Props): ModalHooks => {
  const [form] = Form.useForm<FormInputs>();
  const { modalOpen: openOrUpdate, ...modalHook } = useModal({ onClose });

  const getModalProps = useCallback((): ModalFuncProps => {
    const handleOk = async () => {
      const values = await form.validateFields();
      try {
        await patchModelVersion({
          body: {
            comment: values.description,
            modelName: modelVersion.model.id.toString(),
            name: values.modelVersionName,
          },
          modelName: modelVersion.model.id.toString(),
          versionNum: modelVersion.version,
        });

        await fetchModelVersion();
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to edit model version',
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
          <Form.Item initialValue={modelVersion.name} label="Name" name="modelVersionName">
            <Input />
          </Form.Item>
          <Form.Item initialValue={modelVersion.comment} label="Description" name="description">
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
  }, [
    fetchModelVersion,
    form,
    modelVersion.comment,
    modelVersion.model.id,
    modelVersion.name,
    modelVersion.version,
    onClose,
  ]);

  const modalOpen = useCallback(() => {
    openOrUpdate(getModalProps());
  }, [getModalProps, openOrUpdate]);

  return { modalOpen, ...modalHook };
};

export default useModalModelVersionEdit;
