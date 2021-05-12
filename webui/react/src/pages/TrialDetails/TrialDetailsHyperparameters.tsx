import { List } from 'antd';
import React from 'react';

import Section from 'components/Section';
import { ExperimentBase, TrialDetails } from 'types';
import { isObject } from 'utils/data';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

const TrialDetailsHyperparameters: React.FC<Props> = ({ trial }: Props) => {
  return (
    <Section bodyBorder bodyNoPadding>
      <List
        dataSource={Object.entries(trial.hparams)}
        renderItem={([ label, value ]) => {
          const textValue = isObject(value) ? JSON.stringify(value, null, 2) : value.toString();
          return <List.Item>{label}: {textValue}</List.Item>;
        }}
        size="small"
      />
    </Section>
  );
};

export default TrialDetailsHyperparameters;
