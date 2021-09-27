import { Button, Form, Input } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

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
        <Form.Item
          key={idx}
          name={idx}>
          <Input.Group compact>
            <Form.Item initialValue={pair[0]} name={[ idx, 'label' ]} noStyle>
              <Input placeholder="Enter metadata label" style={{ width: '30%' }} />
            </Form.Item>
            <Form.Item initialValue={pair[1]} name={[ idx, 'value' ]} noStyle>
              <Input placeholder="Enter metadata" style={{ width: '70%' }} />
            </Form.Item>
          </Input.Group>
        </Form.Item>
      );
    });
    for (let i = 0; i < extraRows; i++) {
      md.push(<Form.Item
        key={md.length}
        name={md.length}>
        <Input.Group>
          <Form.Item name={[ md.length, 'label' ]} noStyle>
            <Input placeholder="Enter metadata label" style={{ width: '30%' }} />
          </Form.Item>
          <Form.Item name={[ md.length, 'value' ]} noStyle>
            <Input placeholder="Enter metadata" style={{ width: '70%' }} />
          </Form.Item>
        </Input.Group>
      </Form.Item>);
    }
    return md;
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
      {editing? editableMetadata : <InfoBox rows={staticMetadata} />}
      <Button type="link" onClick={() => setExtraRows(prev => prev+1)}>+ Add Row</Button>
    </Form>
  );
};

export default EditableMetadata;
