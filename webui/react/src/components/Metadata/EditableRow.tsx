import type { DropDownProps, MenuProps } from 'antd';
import { Button, Dropdown, Form, Input } from 'antd';
import { FormListFieldData } from 'antd/lib/form/FormList';
import React, { useMemo } from 'react';

import Icon from 'shared/components/Icon/Icon';
import { ValueOf } from 'shared/types';

import css from './EditableRow.module.scss';

interface Props {
  field?: FormListFieldData;
  initialKey?: string;
  initialValue?: string;
  name: string | number;
  onDelete?: () => void;
}

const EditableRow: React.FC<Props> = ({ name, onDelete, field }: Props) => {
  const menu: DropDownProps['menu'] = useMemo(() => {
    const MenuKey = {
      DeleteMetadataRow: 'delete-metadata-row',
    } as const;

    const funcs = {
      [MenuKey.DeleteMetadataRow]: () => {
        if (onDelete) onDelete();
      },
    };

    const onItemClick: MenuProps['onClick'] = (e) => {
      funcs[e.key as ValueOf<typeof MenuKey>]();
    };

    const menuItems: MenuProps['items'] = [
      { danger: true, key: MenuKey.DeleteMetadataRow, label: 'Delete Row' },
    ];

    return { items: menuItems, onClick: onItemClick };
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
            menu={menu}
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
