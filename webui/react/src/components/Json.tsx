import { List } from 'antd';
import React from 'react';

import { RawJson } from 'types';
import { isObject } from 'utils/data';

interface Props {
  json: RawJson;
}

// TODO can be reused in TrialInfoBox.

const Json: React.FC<Props> = ({ json }: Props) => {

  return (
    <List
      dataSource={Object.entries(json)}
      renderItem={([ label, value ]) => {
        const textValue = isObject(value) ? JSON.stringify(value, null, 2) : value.toString();
        return <List.Item>{label}: {textValue}</List.Item>;
      }}
      size="small"
    />
  );
};

export default Json;
