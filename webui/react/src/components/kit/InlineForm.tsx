import { Form } from 'antd';
import { Rule } from 'antd/es/form';
import React, { useCallback, useEffect, useState } from 'react';

import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import { Primitive } from 'components/kit/internal/types';

import css from './InlineForm.module.scss';

interface Props extends React.PropsWithChildren {
  label: string;
  displayValue?: Primitive;
  initialValue?: Primitive;
  onSubmit: (inputValue: string | number) => Promise<void | Error> | void;
  required?: boolean;
  rules?: Rule[];
  testId?: string;
}

const InlineForm: React.FC<Props> = ({
  label,
  children,
  displayValue = '',
  rules,
  required,
  testId = '',
  onSubmit,
}) => {
  const [isEditing, setIsEditing] = useState(false);
  const [form] = Form.useForm();

  const resetForm = useCallback(() => {
    form.setFieldValue('input', displayValue);
    setIsEditing(false);
  }, [form, displayValue]);

  const submitForm = useCallback(async () => {
    try {
      const formValues = await form.validateFields();
      onSubmit(formValues.input);
    } catch (error) {
      form.setFieldValue('input', displayValue);
    }

    setIsEditing(false);
  }, [form, onSubmit, displayValue]);

  useEffect(() => {
    form.setFieldValue('input', displayValue);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [displayValue]);

  return (
    <Form className={css.formBase} colon={false} form={form} layout="inline" requiredMark={false}>
      <Form.Item
        className={css.formItemInput}
        label={label}
        labelCol={{ span: 0 }}
        name="input"
        required={required}
        rules={rules}
        validateTrigger={['onSubmit']}>
        {isEditing ? (
          children
        ) : (
          <span className={css.readOnlyElement} data-testid={`value-${testId}`}>
            {displayValue}
          </span>
        )}
      </Form.Item>
      <div className={css.buttonsContainer}>
        {isEditing ? (
          <>
            <Button
              data-testid={`submit-${testId}`}
              icon={<Icon name="checkmark" title="confirm" />}
              type="primary"
              onClick={() => {
                form.submit();
                submitForm();
              }}
            />
            <Button
              data-testid={`reset-${testId}`}
              icon={<Icon name="close-small" size="tiny" title="cancel" />}
              type="default"
              onClick={() => resetForm()}
            />
          </>
        ) : (
          <Button
            data-testid={`edit-${testId}`}
            icon={<Icon name="pencil" size="small" title="edit" />}
            type="default"
            onClick={() => setIsEditing(true)}
          />
        )}
      </div>
    </Form>
  );
};

export default InlineForm;
