import React, { useState } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { Modal } from 'components/kit/Modal';
import { patchExperiment } from 'services/api';
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
}

const FORM_ID = 'edit-experiment-form';

const ExperimentEditModalComponent: React.FC<Props> = ({
  experimentName,
  experimentId,
  description,
  fetchExperimentDetails,
}: Props) => {
  const [form] = Form.useForm<FormInputs>();
  const [disabled, setDisabled] = useState<boolean>(true);

  const handleSubmit = async () => {
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

  const handleChange = () => {
    const fields = form.getFieldsError();
    const hasError = fields.some((f) => f.errors.length);
    setDisabled(hasError);
  };

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled,
        handler: handleSubmit,
        text: 'Save',
      }}
      title="Edit Experiment"
      onClose={form.resetFields}>
      <Form
        autoComplete="off"
        form={form}
        id={FORM_ID}
        layout="vertical"
        onFieldsChange={handleChange}>
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
    </Modal>
  );
};

export default ExperimentEditModalComponent;
