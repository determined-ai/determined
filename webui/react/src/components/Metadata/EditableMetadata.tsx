import { Button, Form } from 'antd';
import React, { useCallback, useMemo } from 'react';

import InfoBox, { InfoRow } from 'components/InfoBox';
import { Metadata } from 'types';

import css from './EditableMetadata.module.scss';
import EditableRow from './EditableRow';
interface Props {
  editing?: boolean;
  metadata?: Metadata;
  updateMetadata?: (obj: Metadata) => void;
}

const EditableMetadata: React.FC<Props> = ({ metadata = {}, editing, updateMetadata }: Props) => {
  const [ metadataRows, metadataList ] = useMemo(() => {
    const { rows, list } = Object.entries(metadata).reduce((acc, [ key, value ]) => {
      acc.rows.push({ content: value, label: key });
      acc.list.push({ key, value });
      return acc;
    }, { list: [] as {key: string, value: string}[], rows: [] as InfoRow[] });
    if (list.length === 0) list.push({ key: '', value: '' });
    return [ rows, list ];
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
    <Form initialValues={{ metadata: metadataList }} onValuesChange={onValuesChange}>
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
        </>) : <InfoBox rows={metadataRows} />}
    </Form>
  );
};

export default EditableMetadata;
