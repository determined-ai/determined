import React, { useCallback, useMemo } from 'react';

import InfoBox, { InfoRow } from 'components/InfoBox';
import Button from 'components/kit/Button';
import Form from 'components/kit/Form';
import { Metadata } from 'types';

import css from './EditableMetadata.module.scss';
import EditableRow from './EditableRow';

export const ADD_ROW_TEXT = '+ Add Row';

interface Props {
  editing?: boolean;
  metadata?: Metadata;
  updateMetadata?: (obj: Metadata) => void;
}

const EditableMetadata: React.FC<Props> = ({ metadata = {}, editing, updateMetadata }: Props) => {
  const [metadataRows, metadataList] = useMemo(() => {
    const { rows, list } = Object.entries(metadata).reduce(
      (acc, [key, value]) => {
        acc.rows.push({ content: value, label: key });
        acc.list.push({ key, value });
        return acc;
      },
      { list: [] as { key: string; value: string }[], rows: [] as InfoRow[] },
    );
    if (list.length === 0) list.push({ key: '', value: '' });
    return [rows, list];
  }, [metadata]);

  const onValuesChange = useCallback(
    (_changedValues: unknown, values: { metadata: Metadata[] }) => {
      const newMetadata = values.metadata.reduce((acc, row) => {
        if (row.value === undefined) {
          row.value = '';
        }
        if (row?.key) acc[row.key] = row.value;
        return acc;
      }, {} as Metadata);

      updateMetadata?.(newMetadata);
    },
    [updateMetadata],
  );

  return (
    <Form initialValues={{ metadata: metadataList }} onValuesChange={onValuesChange}>
      {editing ? (
        <>
          <div className={css.titleRow}>
            <span>Key</span>
            <span>Value</span>
          </div>
          <Form.List name="metadata">
            {(fields, { add, remove }) => (
              <>
                {fields.map((field) => (
                  <EditableRow
                    key={field.key}
                    name={field.name}
                    onDelete={fields.length > 1 ? () => remove(field.name) : undefined}
                  />
                ))}
                <Button type="link" onClick={add}>
                  {ADD_ROW_TEXT}
                </Button>
              </>
            )}
          </Form.List>
        </>
      ) : (
        <InfoBox rows={metadataRows} />
      )}
    </Form>
  );
};

export default EditableMetadata;
