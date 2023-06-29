import { Form } from 'antd';
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
    <Form
      className={css.formBase}
      colon={false}
      form={form}
      initialValues={{ layout: 'inline' }}
      layout="inline">
      <Form.Item
        className={css.formItemInput}
        initialValue={inputValue}
        label={label}
        labelCol={{ span: 0 }}
        name="input"
        required={required}>
        {isEditing ? inputElement : <span className={css.readOnlyElement}>{inputValue}</span>}
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
