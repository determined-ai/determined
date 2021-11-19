import { Button, Dropdown, Form, Input, Menu } from 'antd';
import { FormListFieldData } from 'antd/lib/form/FormList';
import React, { useCallback, useMemo } from 'react';

import { RecordKey } from 'types';

import css from './EditableMetadata.module.scss';
import Icon from './Icon';
import InfoBox, { InfoRow } from './InfoBox';

export type Metadata = Record<RecordKey, string>;
interface Props {
  editing?: boolean;
  metadata?: Metadata;
  updateMetadata?: (obj: Metadata) => void;
}

const EditableMetadata: React.FC<Props> = ({ metadata = {}, editing, updateMetadata }: Props) => {
  const staticMetadata: InfoRow[] = useMemo(() => {
    return Object.entries(metadata).map(([ key, value ]) => {
      return ({ content: value, label: key });
    });
  }, [ metadata ]);

  const metadataArray = useMemo(() => {
    const array = Object.entries(metadata).map(([ key, value ]) => {
      return { key, value };
    });
    if (array.length === 0) {
      array.push({ key: '', value: '' });
    }
    return array;
  }, [ metadata ]);

  const onValuesChange = useCallback((
    _changedValues,
    values: {metadata: Metadata[]},
  ) => {
    const newMetadata = (Object.fromEntries(Object.values(values.metadata).map(pair => {
      if (pair == null) return [ '', '' ];
      if (pair.key == null) pair.key = '';
      if (pair.value == null) pair.value = '';
      return [ pair.key, pair.value ];
    })));
    delete newMetadata[''];

    updateMetadata?.(newMetadata);
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

export default EditableMetadata;
