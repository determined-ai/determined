import { Button, Dropdown, Form, Input, Menu } from 'antd';
import { FormListFieldData } from 'antd/lib/form/FormList';
import React, { useCallback, useMemo } from 'react';

import css from './EditableMetadata.module.scss';
import Icon from './Icon';
import InfoBox, { InfoRow } from './InfoBox';

interface Props {
  editing?: boolean;
  metadata: Record<string, string>;
  updateMetadata?: (obj: Record<string, string>) => void;
}

const EditableMetadata: React.FC<Props> = ({ metadata, editing, updateMetadata }: Props) => {
  const staticMetadata: InfoRow[] = useMemo(() => {
    return Object.entries(metadata || {}).map((pair) => {
      return ({ content: pair[1], label: pair[0] });
    });
  }, [ metadata ]);

  const metadataArray = useMemo(() => {
    return [ ...Object.entries(metadata || { }).map(entry => {
      return { key: entry[0], value: entry[1] };
    }), { key: '', value: '' } ];
  }, [ metadata ]);

  const onValuesChange = useCallback((
    _changedValues,
    values: {metadata: Record<string, string>[]},
  ) => {
    const md = (Object.fromEntries(Object.values(values.metadata).map(pair => {
      if (pair == null) return [ '', '' ];
      if (pair.key == null) pair.key = '';
      if (pair.value == null) pair.value = '';
      return [ pair.key, pair.value ];
    })));
    delete md[''];

    updateMetadata?.(md);
  }, [ updateMetadata ]);

  return (
    <Form initialValues={{ metadata: metadataArray }} onValuesChange={onValuesChange}>
      {editing ? (
        <>
          <div className={css.titleRow}>
            <span>Key</span><span>Value</span>
          </div>
          <Form.List name="metadata">
            {(fields, { add, remove }) => (
              <>
                {fields.map(field => (
                  <EditableRow
                    key={field.key}
                    name={field.name}
                    onDelete={() => remove(field.name)} />
                ))}
                <Button
                  className={css.addRow}
                  type="link"
                  onClick={add}>+ Add Row</Button>
              </>)}
          </Form.List>
        </>) : <InfoBox rows={staticMetadata} />}
    </Form>
  );
};

interface EditableRowProps {
  field?: FormListFieldData;
  initialKey?: string;
  initialValue?: string;
  name: string | number;
  onDelete?: () => void;
}

const EditableRow: React.FC<EditableRowProps> = (
  { name, onDelete, field }: EditableRowProps,
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
            <Menu.Item danger onClick={onDelete}>Delete Row</Menu.Item>
          </Menu>
        )}>
        <Button type="text">
          <Icon name="overflow-vertical" size="tiny" />
        </Button>
      </Dropdown>}
    </Input.Group>

  </Form.Item>;
};

export default EditableMetadata;
