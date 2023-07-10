import { Form, FormProps } from 'antd';
import { Rule } from 'antd/es/form';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';

import css from './InlineForm.module.scss';

type InlineForm<T> = (props: Props<T>) => JSX.Element;

interface Props<T> extends React.PropsWithChildren, Omit<FormProps, 'children'> {
  label?: string;
  value?: T; // used to turn the Form.Item as controlled input
  initialValue: T;
  onSubmit?: (inputValue: T) => Promise<void | Error> | void;
  required?: boolean;
  isPassword?: boolean;
  rules?: Rule[];
  testId?: string;
}

function InlineForm<T>({
  label,
  children,
  initialValue,
  value,
  isPassword = false,
  rules,
  required,
  testId = '',
  onSubmit,
  ...formProps
}: Props<T>): JSX.Element {
  const [isEditing, setIsEditing] = useState(false);
  const [previousValue, setPreviousValue] = useState<T>(initialValue);
  const [form] = Form.useForm();
  const shouldColapseText = useMemo(() => String(initialValue).length >= 45, [initialValue]); // prevents layout breaking, specially if using Input.TextArea.
  const inputCurrentValue = Form.useWatch('input', form);
  const readOnlyText = useMemo(() => {
    let textValue = String(value ?? initialValue);
    if (value === undefined) {
      if (inputCurrentValue !== undefined && inputCurrentValue !== initialValue)
        textValue = inputCurrentValue;
    }

    if (isPassword) return textValue.replace(/\S/g, '*');
    if (shouldColapseText) return textValue.slice(0, 50).concat('...');

    return textValue;
  }, [shouldColapseText, value, initialValue, isPassword, inputCurrentValue]);

  const resetForm = useCallback(() => {
    form.resetFields();
    form.setFieldValue('input', previousValue);
    setIsEditing(false);
  }, [form, previousValue]);

  const submitForm = useCallback(async () => {
    try {
      const formValues = await form.validateFields();
      onSubmit?.(formValues.input);

      setPreviousValue(formValues.input);
      setIsEditing(false);
    } catch (error) {
      form.setFieldValue('input', initialValue);
    }
  }, [form, onSubmit, initialValue]);

  useEffect(() => {
    if (value !== undefined) {
      form.setFieldValue('input', value);
    } else {
      form.setFieldValue('input', initialValue);
    }
  }, [initialValue, value, form]);

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
        initialValue={initialValue}
        label={label}
        labelCol={{ span: 0 }}
        name="input"
        required={required}
        rules={rules}
        validateTrigger={['onSubmit', 'onChange']}>
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
                if (form.getFieldError('input').length) return;

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
}

export default InlineForm;
