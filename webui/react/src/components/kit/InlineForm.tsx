import { Form } from 'antd';
import { Rule } from 'antd/es/form';
import React, { useCallback, useEffect, useState } from 'react';

import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import { Primitive } from 'components/kit/internal/types';

import css from './InlineForm.module.scss';

interface Props extends React.PropsWithChildren {
  label: string;
  inputValue?: Primitive;
  onSubmit: (inputValue: string | number) => Promise<void | Error> | void;
  required?: boolean;
  forceEdit?: boolean; // in case we want to start as "edit mode"/isEditing state === true
  rules?: Rule[];
  testId?: string;
}

const InlineForm: React.FC<Props> = ({
  label,
  forceEdit = false,
  children,
  inputValue = '',
  rules,
  required,
  testId = '',
  onSubmit,
}) => {
  const [isEditing, setIsEditing] = useState(forceEdit);
  const [form] = Form.useForm();

  const resetForm = useCallback(() => {
    form.setFieldValue('input', inputValue);
    setIsEditing(false);
  }, [form, inputValue]);

  const submitForm = useCallback(async () => {
    try {
      const formValues = await form.validateFields();

      onSubmit(formValues.input);
    } catch (error) {
      form.setFieldValue('input', inputValue);
    }

    setIsEditing(false);
  }, [form, onSubmit, inputValue]);

  useEffect(() => {
    const fieldValue = form.getFieldValue('input');

    if (fieldValue !== inputValue) form.setFieldValue('input', inputValue);
  }, [form, inputValue]);

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
        label={label}
        labelCol={{ span: 0 }}
        name="input"
        required={required}
        rules={rules}>
        {isEditing ? (
          children
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
                htmlType="submit"
                icon={<Icon name="checkmark" title="confirm" />}
                type="primary"
                onClick={() => {
                  form.submit();
                  submitForm();
                }}
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
