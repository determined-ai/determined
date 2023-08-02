import React, { useCallback } from 'react';

import Button from 'components/kit/Button';
import Dropdown from 'components/kit/Dropdown';
import Form, { FormListFieldData } from 'components/kit/Form';
import Icon from 'components/kit/Icon';
import Input from 'components/kit/Input';

import css from './EditableRow.module.scss';

export const METADATA_KEY_PLACEHOLDER = 'Enter metadata label';
export const METADATA_VALUE_PLACEHOLDER = 'Enter metadata value';
export const DELETE_ROW_LABEL = 'Delete Row';

const DROPDOWN_MENU = [{ danger: true, key: DELETE_ROW_LABEL, label: DELETE_ROW_LABEL }];

interface Props {
  field?: FormListFieldData;
  initialKey?: string;
  initialValue?: string;
  jsonValue?: string;
  name: string | number;
  onDelete?: () => void;
}

const EditableRow: React.FC<Props> = ({ jsonValue, name, onDelete, field }: Props) => {
  const handleDropdown = useCallback(
    (key: string) => {
      switch (key) {
        case DELETE_ROW_LABEL:
          onDelete?.();
          break;
      }
    },
    [onDelete],
  );

  return (
    <Form.Item {...field} name={name} noStyle>
      <Input.Group className={css.row} compact>
        <Form.Item name={[name, 'key']} noStyle>
          <Input disabled={!!jsonValue} placeholder={METADATA_KEY_PLACEHOLDER} />
        </Form.Item>
        <Form.Item name={jsonValue ? '' : [name, 'value']} noStyle>
          <Input disabled={!!jsonValue} placeholder={jsonValue || METADATA_VALUE_PLACEHOLDER} />
        </Form.Item>
        {onDelete && (
          <Dropdown menu={DROPDOWN_MENU} onClick={handleDropdown}>
            <Button
              aria-label="action"
              icon={<Icon name="overflow-vertical" size="tiny" title="Action menu" />}
            />
          </Dropdown>
        )}
      </Input.Group>
    </Form.Item>
  );
};

export default EditableRow;
