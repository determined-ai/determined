import { Button, Form } from 'antd';
import React, { useCallback, useMemo } from 'react';

import InfoBox, { InfoRow } from 'components/InfoBox';
import { RecordKey } from 'types';

import css from './EditableMetadata.module.scss';
import EditableRow from './EditableRow';

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
                    onDelete={fields.length > 1 ? () => remove(field.name) : undefined} />
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

export default EditableMetadata;
