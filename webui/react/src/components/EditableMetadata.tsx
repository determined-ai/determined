import { Button, Form, Input } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import css from './EditableMetadata.module.scss';
import InfoBox, { InfoRow } from './InfoBox';

interface Props {
  editing: boolean;
  metadata: Record<string, string>;
  updateMetadata?: (obj: Record<string, string>) => void;
}

const EditableMetadata: React.FC<Props> = ({ metadata, editing, updateMetadata }: Props) => {
  const [ extraRows, setExtraRows ] = useState(1);
  const staticMetadata: InfoRow[] = useMemo(() => {
    return Object.entries(metadata || {}).map((pair) => {
      return ({ content: pair[1], label: pair[0] });
    });
  }, [ metadata ]);

  const editableMetadata = useMemo(() => {
    const md = Object.entries(metadata || { }).map((pair, idx) => {
      return (
        <EditableRow initialKey={pair[0]} initialValue={pair[1]} key={idx} name={idx} />
      );
    });
    for (let i = 0; i < extraRows; i++) {
      md.push(<EditableRow key={md.length} name={md.length} />);
    }
    return (
      <>
        <div className={css.row}><span>Key</span><span>Value</span></div>
        {md}
      </>
    );
  }, [ metadata, extraRows ]);

  const onValuesChange = useCallback((_changedValues, values: Record<string, string>[]) => {
    const md = (Object.fromEntries(Object.values(values).map(pair => {
      if (pair == null) return [ '', '' ];
      if (pair.label == null) pair.label = '';
      if (pair.value == null) pair.value = '';
      return [ pair.label, pair.value ];
    })));
    delete md[''];
    updateMetadata?.(md);
  }, [ updateMetadata ]);

  return (
    <Form onValuesChange={onValuesChange}>
      {editing ? editableMetadata : <InfoBox rows={staticMetadata} />}
      {editing ?
        <Button
          className={css.addRow}
          type="link"
          onClick={() => setExtraRows(prev => prev+1)}>+ Add Row</Button>
        : null }
    </Form>
  );
};

interface EditableRowProps {
  initialKey?: string;
  initialValue?: string;
  name: string | number;
}

const EditableRow: React.FC<EditableRowProps> = (
  { initialKey, initialValue, name }: EditableRowProps,
) => {
  return <Form.Item
    name={name}
    noStyle>
    <Input.Group className={css.row} compact>
      <Form.Item initialValue={initialKey} name={[ name, 'label' ]} noStyle>
        <Input placeholder="Enter metadata label" />
      </Form.Item>
      <Form.Item initialValue={initialValue} name={[ name, 'value' ]} noStyle>
        <Input placeholder="Enter metadata" />
      </Form.Item>
    </Input.Group>
  </Form.Item>;
};

export default EditableMetadata;
