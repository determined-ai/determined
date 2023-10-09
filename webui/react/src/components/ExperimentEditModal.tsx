import React, { useId, useState } from 'react';

import Form, { hasErrors } from 'components/kit/Form';
import Input from 'components/kit/Input';
import { Modal } from 'components/kit/Modal';
import { patchExperiment } from 'services/api';
import { ExperimentItem } from 'types';
import handleError from 'utils/error';

export const NAME_LABEL = 'Name';
export const DESCRIPTION_LABEL = 'Description';
export const BUTTON_TEXT = 'Save';

type FormInputs = {
  description: string;
  experimentName: string;
};

interface Props {
  description: string;
  experimentId: number;
  experimentName: string;
  onEditComplete: (data: Partial<ExperimentItem>) => void;
}

const FORM_ID = 'edit-experiment-form';

const ExperimentEditModalComponent: React.FC<Props> = ({
  experimentName,
  experimentId,
  description,
  onEditComplete,
}: Props) => {
  const idPrefix = useId();
  const [form] = Form.useForm<FormInputs>();
  const [disabled, setDisabled] = useState<boolean>(true);

  const handleSubmit = async () => {
    const formData = await form.validateFields();
    try {
      await patchExperiment({
        body: { description: formData.description, name: formData.experimentName },
        experimentId,
      });
      onEditComplete({ description: formData.description, name: formData.experimentName });
    } catch (e) {
      handleError(e, {
        publicMessage: 'Unable to update experiment',
        silent: false,
      });
    }
  };

  const handleChange = () => {
    setDisabled(hasErrors(form));
  };

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled,
        form: idPrefix + FORM_ID,
        handleError,
        handler: handleSubmit,
        text: BUTTON_TEXT,
      }}
      title="Edit Experiment"
      onClose={form.resetFields}>
      <Form
        autoComplete="off"
        form={form}
        id={idPrefix + FORM_ID}
        layout="vertical"
        onFieldsChange={handleChange}>
        <Form.Item
          initialValue={experimentName}
          label={NAME_LABEL}
          name="experimentName"
          rules={[
            {
              max: 128,
              message: 'Name must be 1 ~ 128 characters',
              required: true,
              whitespace: true,
            },
          ]}>
          <Input />
        </Form.Item>
        <Form.Item initialValue={description} label={DESCRIPTION_LABEL} name="description">
          <Input.TextArea />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ExperimentEditModalComponent;
