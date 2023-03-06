import type { DropDownProps, MenuProps } from 'antd';
import { Dropdown } from 'antd';
import React, { useMemo } from 'react';

import Button from 'components/kit/Button';
import Form, { FormListFieldData } from 'components/kit/Form';
import Input from 'components/kit/Input';
import Icon from 'shared/components/Icon/Icon';
import { ValueOf } from 'shared/types';

import css from './EditableRow.module.scss';

export const METADATA_KEY_PLACEHOLDER = 'Enter metadata label';
export const METADATA_VALUE_PLACEHOLDER = 'Enter metadata value';
export const DELETE_ROW_LABEL = 'Delete Row';

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
        onDelete?.();
      },
    };

    const onItemClick: MenuProps['onClick'] = (e) => {
      funcs[e.key as ValueOf<typeof MenuKey>]();
    };

    const menuItems: MenuProps['items'] = [
      { danger: true, key: MenuKey.DeleteMetadataRow, label: DELETE_ROW_LABEL },
    ];

    return { items: menuItems, onClick: onItemClick };
  }, [onDelete]);

  return (
    <Form.Item {...field} name={name} noStyle>
      <Input.Group className={css.row} compact>
        <Form.Item name={[name, 'key']} noStyle>
          <Input placeholder={METADATA_KEY_PLACEHOLDER} />
        </Form.Item>
        <Form.Item name={[name, 'value']} noStyle>
          <Input placeholder={METADATA_VALUE_PLACEHOLDER} />
        </Form.Item>
        {onDelete && (
          <Dropdown
            className={css.overflow}
            getPopupContainer={(triggerNode) => triggerNode}
            menu={menu}
            trigger={['click']}>
            <Button
              aria-label="action"
              ghost
              icon={<Icon name="overflow-vertical" size="tiny" />}
            />
          </Dropdown>
        )}
      </Input.Group>
    </Form.Item>
  );
};

export default EditableRow;
