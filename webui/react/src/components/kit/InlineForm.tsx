import React, { useCallback, useState } from 'react';

import Button from 'components/kit/Button';
import Form from 'components/kit/Form';
import Icon from 'components/kit/Icon';

import css from './InlineForm.module.scss';

interface Props {
  label: string;
  inputValue?: string | number;
  onSubmit: (inputValue: string | number) => Promise<void> | void;
  inputElement: React.ReactNode;
  required?: boolean;
  forceEdit?: boolean;
}

const InlineForm: React.FC<Props> = ({
  label,
  required,
  forceEdit = false,
  inputElement,
  inputValue = '',
  onSubmit,
}) => {
  const [isEditing, setIsEditing] = useState(forceEdit);
  const [form] = Form.useForm();

  const resetForm = useCallback(() => {
    form.setFieldValue('input', inputValue);
    setIsEditing(false);
  }, [inputValue, form]);

  const submitForm = useCallback(() => {
    onSubmit(form.getFieldValue('input'));
    setIsEditing(false);
  }, [form, onSubmit]);

  return (
    <Form className={css.formBase} form={form} initialValues={{ layout: 'inline' }} layout="inline">
      <Form.Item
        className={css.formItemInput}
        initialValue={inputValue}
        label={label}
        labelCol={{ span: 0 }}
        name="input"
        required={required}>
        {React.isValidElement(inputElement) &&
          React.cloneElement(inputElement, { ...inputElement.props, disabled: !isEditing })}
      </Form.Item>
      <div className={css.buttonsContainer}>
        {isEditing ? (
          <>
            <Form.Item>
              <Button
                icon={<Icon name="checkmark" title="confirm" />}
                type="primary"
                onClick={() => submitForm()}
              />
            </Form.Item>
            <Form.Item>
              <Button
                icon={<Icon name="close-small" size="tiny" title="cancel" />}
                type="default"
                onClick={() => resetForm()}
              />
            </Form.Item>
          </>
        ) : (
          <Form.Item>
            <Button
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
