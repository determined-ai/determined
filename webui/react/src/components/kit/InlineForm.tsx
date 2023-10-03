import { Form, FormProps } from 'antd';
import { Rule } from 'antd/es/form';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';

import css from './InlineForm.module.scss';

type InlineForm<T> = (props: Props<T>) => JSX.Element;

interface Props<T> extends React.PropsWithChildren, Omit<FormProps, 'children'> {
  label?: React.ReactNode;
  value?: T; // used to turn the Form.Item as controlled input
  initialValue: T;
  onSubmit?: (inputValue: T) => Promise<void | Error> | void;
  required?: boolean;
  isPassword?: boolean;
  rules?: Rule[];
  testId?: string;
  valueFormatter?: (value: T) => string;
  open?: boolean; // used to set `isEditing` as a controlled value
  onEdit?: () => void;
  onCancel?: () => void;
}

function InlineForm<T>({
  label,
  children,
  valueFormatter,
  initialValue,
  value,
  isPassword = false,
  rules,
  required,
  testId = '',
  onSubmit,
  onEdit,
  onCancel,
  open,
  ...formProps
}: Props<T>): JSX.Element {
  const [isEditing, setIsEditing] = useState(false);
  const [hasFormError, setHasFormError] = useState(false);
  const [previousValue, setPreviousValue] = useState<T>(initialValue); // had to set a state due to uncontrolled form reseting to the initialValue instead of previous value
  const [form] = Form.useForm();
  const shouldCollapseText = useMemo(() => String(initialValue).length >= 45, [initialValue]); // prevents layout breaking, specially if using Input.TextArea.
  const inputCurrentValue = Form.useWatch('input', form);
  const readOnlyText = useMemo(() => {
    let textValue = valueFormatter
      ? valueFormatter(value ?? initialValue)
      : String(value ?? initialValue);
    if (value === undefined) {
      if (inputCurrentValue !== undefined && inputCurrentValue !== initialValue)
        textValue = valueFormatter ? valueFormatter(inputCurrentValue) : inputCurrentValue;
    }

    if (isPassword) return textValue.replace(/\S/g, '*');
    if (shouldCollapseText) return textValue.slice(0, 50).concat('...');
    return textValue;
  }, [shouldCollapseText, value, initialValue, isPassword, inputCurrentValue, valueFormatter]);

  useEffect(() => {
    if (value !== undefined) {
      form.setFieldValue('input', value);
    } else {
      form.setFieldValue('input', initialValue);
    }
  }, [initialValue, value, form]);

  useEffect(() => {
    if (open !== undefined) setIsEditing(open);
  }, [open]);

  React.useEffect(() => {
    // Reacts to the input validation to enable/disable the "submit button"
    let mounted = true;
    if (isEditing) {
      (async () => {
        try {
          await form.validateFields(['input'])
          if (mounted) setHasFormError(false)
        } catch {
          if (mounted) setHasFormError(true)
        }
      })()
    }
    return () => {
      mounted = false;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [inputCurrentValue, isEditing, form]);

  const handleEdit = useCallback(() => {
    if (open === undefined) setIsEditing(true);
    onEdit?.();
  }, [open, onEdit]);

  const handleCancel = useCallback(() => {
    form.resetFields();
    form.setFieldValue('input', previousValue);

    if (open === undefined) setIsEditing(false);
    onCancel?.();
  }, [open, onCancel, form, previousValue]);

  const handleConfirm = useCallback(async () => {
    if (form.getFieldError('input').length) return;
    form.submit();
    try {
      const formValues = await form.validateFields();
      setPreviousValue(formValues.input);

      if (open === undefined) setIsEditing(false);
      onSubmit?.(formValues.input);
    } catch (error) {
      form.setFieldValue('input', initialValue);
    }
  }, [form, initialValue, open, onSubmit]);

  return (
    <Form
      className={css.formBase}
      colon={false}
      form={form}
      layout="inline"
      requiredMark={false}
      {...formProps}>
      <label>{label}</label>
      <Form.Item
        className={css.formItemInput}
        initialValue={initialValue}
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
              disabled={hasFormError}
              icon={<Icon name="checkmark" title="confirm" />}
              type="primary"
              onClick={handleConfirm}
            />
            <Button
              data-testid={`reset-${testId}`}
              icon={<Icon name="close-small" size="tiny" title="cancel" />}
              type="default"
              onClick={handleCancel}
            />
          </>
        ) : (
          <Button
            data-testid={`edit-${testId}`}
            icon={<Icon name="pencil" size="small" title="edit" />}
            type="default"
            onClick={handleEdit}
          />
        )}
      </div>
    </Form>
  );
}

export default InlineForm;
