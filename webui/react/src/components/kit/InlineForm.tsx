import { Form } from 'antd';
import { Rule } from 'antd/es/form';
import React, { useCallback, useState } from 'react';

import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';

import css from './InlineForm.module.scss';

interface Props {
  label: string;
  inputValue?: string | number;
  onSubmit: (inputValue: string | number) => Promise<void | Error> | void;
  inputElement: React.ReactNode;
  required?: boolean;
  forceEdit?: boolean;
  rules?: Rule[];
  testId?: string;
}

const InlineForm: React.FC<Props> = ({
  label,

  forceEdit = false,
  inputElement,
  inputValue = '',
  rules,
  required,
  testId = '',
  onSubmit,
}) => {
  const [isEditing, setIsEditing] = useState(forceEdit);
  const [form] = Form.useForm();

  const resetForm = useCallback(() => {
    form.resetFields();
    setIsEditing(false);
  }, [form]);

  const submitForm = useCallback(async () => {
    try {
      const formValues = await form.validateFields();

      onSubmit(formValues.input);
    } catch (error) {
      form.resetFields();
    }

    setIsEditing(false);
  }, [form, onSubmit]);

  return (
    <Form
      className={css.formBase}
      colon={false}
      form={form}
      initialValues={{ layout: 'inline' }}
      layout="inline"
      requiredMark={false}>
      <Form.Item
        className={css.formItemInput}
        initialValue={inputValue}
        label={label}
        labelCol={{ span: 0 }}
        name="input"
        required={required}
        rules={rules}>
        {isEditing ? (
          inputElement
        ) : (
          <span className={css.readOnlyElement} data-testid={`value-${testId}`}>
            {inputValue}
          </span>
        )}
      </Form.Item>
      <div className={css.buttonsContainer}>
        {isEditing ? (
          <>
            <Form.Item>
              <Button
                data-testid={`submit-${testId}`}
                icon={<Icon name="checkmark" title="confirm" />}
                type="primary"
                onClick={() => submitForm()}
              />
            </Form.Item>
            <Form.Item>
              <Button
                data-testid={`reset-${testId}`}
                icon={<Icon name="close-small" size="tiny" title="cancel" />}
                type="default"
                onClick={() => resetForm()}
              />
            </Form.Item>
          </>
        ) : (
          <Form.Item>
            <Button
              data-testid={`edit-${testId}`}
              icon={<Icon name="pencil" size="small" title="edit" />}
              type="default"
              onClick={() => setIsEditing(true)}
            />
          </Form.Item>
        )}
      </div>
    </Form>
  );
};

export default InlineForm;
