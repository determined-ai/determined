import { Form, FormProps } from 'antd';
import { Rule } from 'antd/es/form';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import { Primitive } from 'components/kit/internal/types';

import css from './InlineForm.module.scss';

interface Props extends React.PropsWithChildren, Omit<FormProps, 'children'> {
  label?: string;
  displayValue?: Primitive;
  initialValue?: Primitive;
  onSubmit: (inputValue: string | number) => Promise<void | Error> | void;
  required?: boolean;
  isPassword?: boolean;
  rules?: Rule[];
  testId?: string;
}

const InlineForm: React.FC<Props> = ({
  label,
  children,
  displayValue = '',
  isPassword = false,
  rules,
  required,
  testId = '',
  onSubmit,
  ...formProps
}) => {
  const [isEditing, setIsEditing] = useState(false);
  const [form] = Form.useForm();
  const shouldColapseText = useMemo(() => String(displayValue).length >= 45, [displayValue]); // prevents layout breaking, specially if using Input.TextArea.
  const readOnlyText = useMemo(() => {
    if (isPassword) return String(displayValue).replace(/\S/g, '*');
    if (shouldColapseText) return String(displayValue).slice(0, 50).concat('...');

    return displayValue;
  }, [shouldColapseText, displayValue, isPassword]);

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
    <Form
      className={css.formBase}
      colon={false}
      form={form}
      layout="inline"
      requiredMark={false}
      {...formProps}>
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
            {readOnlyText}
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
