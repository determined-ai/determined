import React, { useMemo } from 'react';

import InteractiveTable, {
  ColumnDef,
  InteractiveTableSettings,
} from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import { defaultRowClassName } from 'components/Table/Table';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import Spinner from 'shared/components/Spinner';
import { isObject } from 'shared/utils/data';
import { alphaNumericSorter } from 'shared/utils/sort';
import { TrialDetails } from 'types';

import css from './TrialDetailsHyperparameters.module.scss';
import settingsConfig, { Settings } from './TrialDetailsHyperparameters.settings';

export interface Props {
  pageRef: React.RefObject<HTMLElement>;
  trial: TrialDetails;
}

interface HyperParameter {
  hyperparameter: string;
  value: string;
}

const TrialDetailsHyperparameters: React.FC<Props> = ({ trial, pageRef }: Props) => {
  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);

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
    <div className={css.base}>
      <Spinner spinning={!trial}>
        {trial ? (
          <InteractiveTable
            columns={columns}
            containerRef={pageRef}
            dataSource={dataSource}
            pagination={false}
            rowClassName={defaultRowClassName({ clickable: false })}
            rowKey="hyperparameter"
            settings={settings as InteractiveTableSettings}
            showSorterTooltip={false}
            size="small"
            updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
          />
        ) : (
          <SkeletonTable columns={columns.length} />
        )}
      </Spinner>
    </div>
  );
};

export default TrialDetailsHyperparameters;
