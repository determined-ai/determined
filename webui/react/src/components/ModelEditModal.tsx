import { useId } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { Modal } from 'components/kit/Modal';
import { patchModel } from 'services/api';
import { ModelItem } from 'types';
import handleError from 'utils/error';

type FormInputs = {
  modelName: string;
  description?: string;
};

interface Props {
  fetchModel: () => Promise<void>;
  model: ModelItem;
}

const FORM_ID = 'edit-model-form';

const ModelEditModal = ({ model, fetchModel }: Props): JSX.Element => {
  const idPrefix = useId();
  const [form] = Form.useForm<FormInputs>();

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
  };

  return (
    <Modal
      size="small"
      submit={{ form: idPrefix + FORM_ID, handleError, handler: handleOk, text: 'Save' }}
      title="Edit Model"
      onClose={handleClose}>
      <Form autoComplete="off" form={form} id={idPrefix + FORM_ID} layout="vertical">
        <Form.Item
          initialValue={model.name}
          label="Name"
          name="modelName"
          rules={[{ message: 'Name is required', required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item initialValue={model.description} label="Description" name="description">
          <Input.TextArea />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ModelEditModal;
