import { Popover, Space } from 'antd';
import { CheckboxChangeEvent } from 'antd/es/checkbox';
import React, { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';

import Button from 'components/kit/Button';
import Checkbox from 'components/kit/Checkbox';
import Input from 'components/kit/Input';
import Pivot from 'components/kit/Pivot';
import { V1LocationType } from 'services/api-ts-sdk';
import Spinner from 'shared/components/Spinner';
import { ProjectColumn } from 'types';
import { Loadable } from 'utils/loadable';

import css from './ColumnPickerMenu.module.scss';
import { defaultExperimentColumns } from './columns';

const BANNED_COLUMNS = new Set(['name']);

const removeBannedColumns = (columns: ProjectColumn[]) =>
  columns.filter((col) => !BANNED_COLUMNS.has(col.column));

const locationLabelMap = {
  [V1LocationType.EXPERIMENT]: 'General',
  [V1LocationType.VALIDATIONS]: 'Metrics',
  [V1LocationType.HYPERPARAMETERS]: 'Hyperparameters',
} as const;

interface ColumnMenuProps {
  initialVisibleColumns: string[];
  projectColumns: Loadable<ProjectColumn[]>;
  setVisibleColumns: (newColumns: string[]) => void;
}

interface ColumnTabProps {
  columnState: Record<string, boolean>;
  handleShowSuggested: () => void;
  search: string;
  setSearch: React.Dispatch<React.SetStateAction<string>>;
  setColumnState: React.Dispatch<React.SetStateAction<Record<string, boolean>>>;
  tab: V1LocationType;
  totalColumns: ProjectColumn[];
}

const ColumnPickerTabNF: React.FC<ColumnTabProps> = ({
  columnState,
  setColumnState,
  handleShowSuggested,
  search,
  setSearch,
  tab,
  totalColumns,
}) => {
  const filteredColumns = useMemo(() => {
    const regex = new RegExp(search, 'i');
    return totalColumns.filter(
      (col) => col.location === tab && regex.test(col.displayName || col.column),
    );
  }, [search, totalColumns, tab]);

  const allFilteredColumnsChecked = useMemo(() => {
    return filteredColumns.map((col) => columnState[col.column]).every((col) => col === true);
  }, [columnState, filteredColumns]);

  const handleShowHideAll = useCallback(() => {
    const filteredColumnMap: Record<string, boolean> = filteredColumns.reduce(
      (acc, col) => ({ ...acc, [col.column]: !allFilteredColumnsChecked }),
      {},
    );

    setColumnState((prevCols) => ({ ...prevCols, ...filteredColumnMap }));
  }, [allFilteredColumnsChecked, filteredColumns, setColumnState]);

  const handleColumnChange = useCallback(
    (event: CheckboxChangeEvent) => {
      const [id, checked] = [event.target.id, event.target.checked];
      if (id === undefined) return;
      setColumnState((prevState) => ({ ...prevState, [id]: checked }));
    },
    [setColumnState],
  );

  const handleSearch = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      setSearch(e.target.value);
    },
    [setSearch],
  );

  return (
    <div>
      <Input allowClear autoFocus placeholder="Search" value={search} onChange={handleSearch} />
      {totalColumns.length !== 0 ? (
        <Space className={css.columnList} direction="vertical">
          {filteredColumns.map((col) => (
            <Checkbox
              checked={columnState[col.column] ?? false}
              id={col.column}
              key={col.column}
              onChange={handleColumnChange}>
              {col.displayName || col.column}
            </Checkbox>
          ))}
        </Space>
      ) : (
        <Spinner />
      )}
      <div className={css.actionRow}>
        <Button type="text" onClick={handleShowHideAll}>
          {allFilteredColumnsChecked ? 'Hide' : 'Show'} all
        </Button>
        <Button type="text" onClick={handleShowSuggested}>
          Show suggested
        </Button>
      </div>
    </div>
  );
};

const ColumnPickerMenu: React.FC<ColumnMenuProps> = ({
  projectColumns,
  setVisibleColumns,
  initialVisibleColumns,
}) => {
  const [search, setSearch] = useState('');

  const totalColumns = useMemo(
    () => removeBannedColumns(Loadable.getOrElse([], projectColumns)),
    [projectColumns],
  );

  const [columnState, setColumnState] = useState<Record<string, boolean>>(() =>
    totalColumns.reduce(
      (acc, col) => ({ ...acc, [col.column]: initialVisibleColumns.includes(col.column) }),
      {},
    ),
  );

  useEffect(() => {
    if (Object.keys(columnState).length === 0) return;
    /* eslint-disable @typescript-eslint/no-unused-vars */
    setVisibleColumns(
      Object.entries(columnState)
        .filter(([_, checked]) => checked)
        .map(([column, _]) => column),
    );
    /* eslint-enable @typescript-eslint/no-unused-vars */
  }, [columnState, setVisibleColumns]);

  const handleShowSuggested = useCallback(() => {
    const defaultCols: Set<string> = new Set(defaultExperimentColumns);

    setColumnState((prevColumns) =>
      Object.fromEntries(Object.keys(prevColumns).map((col) => [col, defaultCols.has(col)])),
    );
  }, []);

  return (
    <Popover
      content={
        <div className={css.base}>
          <Pivot
            items={[
              V1LocationType.EXPERIMENT,
              V1LocationType.VALIDATIONS,
              V1LocationType.HYPERPARAMETERS,
            ].map((tab) => ({
              children: (
                <ColumnPickerTabNF
                  columnState={columnState}
                  handleShowSuggested={handleShowSuggested}
                  search={search}
                  setColumnState={setColumnState}
                  setSearch={setSearch}
                  tab={tab}
                  totalColumns={totalColumns}
                />
              ),
              forceRender: true,
              key: tab,
              label: locationLabelMap[tab],
            }))}
          />
        </div>
      }
      placement="bottom"
      trigger="click">
      <Button>Columns</Button>
    </Popover>
  );
};

export default ColumnPickerMenu;
