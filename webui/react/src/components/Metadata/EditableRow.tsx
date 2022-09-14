import type { MenuProps } from 'antd';
import { Button, Dropdown, Form, Input, Menu } from 'antd';
import { FormListFieldData } from 'antd/lib/form/FormList';
import React, { useMemo } from 'react';

import Icon from 'shared/components/Icon/Icon';

import css from './EditableRow.module.scss';

interface Props {
  field?: FormListFieldData;
  initialKey?: string;
  initialValue?: string;
  name: string | number;
  onDelete?: () => void;
}

const EditableRow: React.FC<Props> = ({ name, onDelete, field }: Props) => {
  const menu = useMemo(() => {
    enum MenuKey {
      DELETE_METADATA_ROW = 'delete-metadata-row',
    }

    const funcs = {
      [MenuKey.DELETE_METADATA_ROW]: () => {
        if (onDelete) onDelete();
      },
    };

    const onItemClick: MenuProps['onClick'] = (e) => {
      funcs[e.key as MenuKey]();
    };

    const menuItems: MenuProps['items'] = [
      { danger: true, key: MenuKey.DELETE_METADATA_ROW, label: 'Delete Row' },
    ];

    return <Menu items={menuItems} onClick={onItemClick} />;
  }, [onDelete]);

  return (
    <Form.Item {...field} name={name} noStyle>
      <Input.Group className={css.row} compact>
        <Form.Item name={[name, 'key']} noStyle>
          <Input placeholder="Enter metadata label" />
        </Form.Item>
        <Form.Item name={[name, 'value']} noStyle>
          <Input placeholder="Enter metadata value" />
        </Form.Item>
        {onDelete && (
          <Dropdown
            className={css.overflow}
            getPopupContainer={(triggerNode) => triggerNode}
            overlay={menu}
            trigger={['click']}>
            <Button aria-label="action" type="text">
              <Icon name="overflow-vertical" size="tiny" />
            </Button>
          </Dropdown>
        )}
      </Input.Group>
    </Form.Item>
  );
};

export default EditableRow;
