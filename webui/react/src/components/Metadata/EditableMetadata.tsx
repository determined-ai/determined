import React, { useCallback, useEffect, useMemo } from 'react';

import InfoBox, { InfoRow } from 'components/InfoBox';
import Form from 'components/kit/Form';
import Link from 'components/Link';
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
        const stringedValue = typeof value === 'object' ? JSON.stringify(value) : value;
        acc.rows.push({ content: stringedValue, label: key });
        acc.list.push({ key, value });
        return acc;
      },
      { list: [] as { key: string; value: string | object }[], rows: [] as InfoRow[] },
    );
    if (list.length === 0) list.push({ key: '', value: '' });
    return [rows, list];
  }, [metadata]);

  const onValuesChange = useCallback(
    (_changedValues: { metadata: Metadata[] }, values: { metadata: Metadata[] }) => {
      const mapAndUpdate = (metadata: Metadata[]) => {
        // filtering with Boolean as, upon removing a row, it triggers the onValuesChange with the removed row as an undefined entry.
        const newMetadata = metadata.filter(Boolean).reduce((acc, row) => {
          if (row.value === undefined) {
            row.value = '';
          }
          if (typeof row?.key === 'string') acc[row.key] = row.value;
          return acc;
        }, {} as Metadata);

        updateMetadata?.(newMetadata);
      };

      mapAndUpdate(values.metadata);
    },
    [updateMetadata],
  );

  const [form] = Form.useForm();
  useEffect(() => {
    form.resetFields();
  }, [form, editing]);

  return (
    <Form form={form} initialValues={{ metadata: metadataList }} onValuesChange={onValuesChange}>
      {editing ? (
        <>
          <div className={css.titleRow}>
            <span>Key</span>
            <span>Value</span>
          </div>
          <Form.List name="metadata">
            {(fields, { add, remove }) => (
              <>
                {fields.map((field, idx) => (
                  <EditableRow
                    jsonValue={
                      typeof metadataList[idx]?.value === 'object'
                        ? JSON.stringify(metadataList[idx]?.value || '')
                        : undefined
                    }
                    key={field.key}
                    name={field.name}
                    onDelete={() => remove(field.name)}
                  />
                ))}
                <Link onClick={() => add({ key: '', value: '' })}>{ADD_ROW_TEXT}</Link>
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
