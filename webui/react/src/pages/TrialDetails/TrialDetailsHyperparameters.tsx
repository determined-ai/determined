import React, { useMemo } from 'react';

import Spinner from 'components/kit/Spinner';
import InteractiveTable, { ColumnDef } from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import { defaultRowClassName } from 'components/Table/Table';
import { useSettings } from 'hooks/useSettings';
import { TrialDetails } from 'types';
import { isObject } from 'utils/data';
import { alphaNumericSorter } from 'utils/sort';

import { configForTrial, Settings } from './TrialDetailsHyperparameters.settings';

export interface Props {
  pageRef: React.RefObject<HTMLElement>;
  trial: TrialDetails;
}

interface HyperParameter {
  hyperparameter: string;
  value: string;
}

const TrialDetailsHyperparameters: React.FC<Props> = ({ trial, pageRef }: Props) => {
  const config = useMemo(() => configForTrial(trial?.id), [trial?.id]);
  const { settings, updateSettings } = useSettings<Settings>(config);

  const columns: ColumnDef<HyperParameter>[] = useMemo(
    () => [
      {
        dataIndex: 'hyperparameter',
        defaultSortOrder: 'ascend',
        defaultWidth: 200,
        key: 'hyperparameter',
        sorter: (a: HyperParameter, b: HyperParameter) =>
          alphaNumericSorter(a.hyperparameter, b.hyperparameter),
        title: 'Hyperparameter',
      },
      {
        dataIndex: 'value',
        defaultWidth: 300,
        key: 'value',
        title: 'Value',
      },
    ],
    [],
  );

  const dataSource: HyperParameter[] = useMemo(() => {
    if (trial?.hyperparameters == null) return [];
    return Object.entries(trial.hyperparameters).map(([hyperparameter, value]) => {
      return {
        hyperparameter,
        key: hyperparameter,
        value: isObject(value) ? JSON.stringify(value, null, 2) : String(value),
      };
    });
  }, [trial?.hyperparameters]);

  return (
    <Spinner spinning={!trial}>
      {trial ? (
        <InteractiveTable<HyperParameter, Settings>
          columns={columns}
          containerRef={pageRef}
          dataSource={dataSource}
          pagination={false}
          rowClassName={defaultRowClassName({ clickable: false })}
          rowKey="hyperparameter"
          settings={settings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings}
        />
      ) : (
        <SkeletonTable columns={columns.length} />
      )}
    </Spinner>
  );
};

export default TrialDetailsHyperparameters;
