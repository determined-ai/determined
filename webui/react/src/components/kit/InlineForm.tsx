import React, { useCallback, useMemo, useState } from 'react';

import Button from 'components/kit/Button';
import Form from 'components/kit/Form';
import Icon from 'components/kit/Icon';
import Input from 'components/kit/Input';
import Select, { Option, SelectValue } from 'components/kit/Select';

import css from './Inlineform.module.scss';

type Option = {
  label: string;
  value: SelectValue;
};

interface Props {
  label: string;
  initialInputValue?: string;
  onSubmit?: () => Promise<void>;
  required?: boolean;
  type: 'input' | 'select';
  defaultSelectOption?: SelectValue;
  selectOptions?: Option[];
  selectSearchable?: boolean;
}

const InlineForm: React.FC<Props> = ({
  label,
  required,
  type,
  selectOptions,
  defaultSelectOption,
  selectSearchable,
  initialInputValue = '',
  onSubmit,
}) => {
  const [isEditing, setIsEditing] = useState(() => {
    if (type === 'input' && !initialInputValue) return true;

    return false;
  });
  const [form] = Form.useForm();

  const element = useMemo(() => {
    if (type === 'input') return <Input disabled={!isEditing} />;

    if (!selectOptions) {
      throw new Error("No 'selectOptions' prop present...");
    }

    return (
      <Select
        defaultValue={defaultSelectOption}
        disabled={!isEditing}
        searchable={selectSearchable}>
        {selectOptions.map((opt) => (
          <Option key={opt.value as React.Key} value={opt.value}>
            {opt.label}
          </Option>
        ))}
      </Select>
    );
  }, [isEditing, type, selectOptions, defaultSelectOption, selectSearchable]);

  const resetForm = useCallback(() => {
    form.setFieldValue('input', type === 'input' ? initialInputValue : defaultSelectOption);
    setIsEditing(false);
  }, [type, initialInputValue, defaultSelectOption, form]);

  return (
    <Form className={css.formBase} form={form} initialValues={{ layout: 'inline' }} layout="inline">
      <Form.Item
        className={css.formItemInput}
        label={label}
        labelCol={{ span: 0 }}
        name="input"
        required={required}>
        {element}
      </Form.Item>
      <div className={css.buttonsContainer}>
        {isEditing ? (
          <>
            <Form.Item>
              <Button
                icon={<Icon name="checkmark" title="confirm" />}
                type="primary"
                onClick={() => onSubmit?.()}
              />
            </Form.Item>
            <Form.Item className={css.cancelButton}>
              <Button
                icon={<Icon name="close-small" size="tiny" title="cancel" />}
                type="default"
                onClick={() => resetForm()}
              />
            </Form.Item>
          </>
        ) : (
          <Form.Item className={css.cancelButton}>
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
