import { Button, Dropdown, Form, Input, Menu } from 'antd';
import { FormListFieldData } from 'antd/lib/form/FormList';
import React from 'react';

import Icon from 'components/Icon';

import css from './EditableRow.module.scss';

interface Props {
  field?: FormListFieldData;
  initialKey?: string;
  initialValue?: string;
  name: string | number;
  onDelete?: () => void;
}

const EditableRow: React.FC<Props> = (
  { name, onDelete, field }: Props,
) => {
  return <Form.Item
    {...field}
    name={name}
    noStyle>
    <Input.Group className={css.row} compact>
      <Form.Item name={[ name, 'key' ]} noStyle>
        <Input placeholder="Enter metadata label" />
      </Form.Item>
      <Form.Item name={[ name, 'value' ]} noStyle>
        <Input placeholder="Enter metadata value" />
      </Form.Item>
      {onDelete && <Dropdown
        className={css.overflow}
        overlay={(
          <Menu>
            <Menu.Item danger key="delete-metadata-row" onClick={onDelete}>Delete Row</Menu.Item>
          </Menu>
        )}
        trigger={[ 'click' ]}>
        <Button type="text">
          <Icon name="overflow-vertical" size="tiny" />
        </Button>
      </Dropdown>}
    </Input.Group>
  </Form.Item>;
};

export default EditableRow;
