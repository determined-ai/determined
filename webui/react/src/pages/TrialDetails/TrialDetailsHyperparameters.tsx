import { ColumnType } from 'antd/es/table';
import React, { useMemo } from 'react';

import ResponsiveTable from 'components/ResponsiveTable';
import { ExperimentBase, TrialDetails } from 'types';
import { isObject } from 'utils/data';
import { alphanumericSorter } from 'utils/sort';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

interface HyperParameter {
  hyperparameter: string,
  value: string,
}

const TrialDetailsHyperparameters: React.FC<Props> = ({ trial }: Props) => {
  const columns: ColumnType<HyperParameter>[] = useMemo(() => [
    {
      dataIndex: 'hyperparameter',
      defaultSortOrder: 'ascend',
      key: 'hyperparameter',
      sorter: (a: HyperParameter, b: HyperParameter) =>
        alphanumericSorter(a.hyperparameter, b.hyperparameter),
      title: 'Hyperparameter',
    },
    {
      dataIndex: 'value',
      key: 'value',
      title: 'Value',
    },
  ], []);
  const dataSource: HyperParameter[] = useMemo(() => {
    return Object.entries(trial.hparams).map(([ hyperparameter, value ]) => {
      return {
        hyperparameter,
        key: hyperparameter,
        value: isObject(value) ? JSON.stringify(value, null, 2) : value.toString(),
      };
    });
  }, [ trial.hparams ]);

  return (
    <ResponsiveTable
      columns={columns}
      dataSource={dataSource}
      pagination={false}
    />
  );
};

export default TrialDetailsHyperparameters;
