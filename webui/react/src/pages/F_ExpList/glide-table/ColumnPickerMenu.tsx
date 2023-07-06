import { Popover, Space } from 'antd';
import { CheckboxChangeEvent } from 'antd/es/checkbox';
import React, { ChangeEvent, useCallback, useMemo, useState } from 'react';

import Button from 'components/kit/Button';
import Checkbox from 'components/kit/Checkbox';
import Empty from 'components/kit/Empty';
import Icon from 'components/kit/Icon';
import Input from 'components/kit/Input';
import Pivot from 'components/kit/Pivot';
import Spinner from 'components/Spinner';
import { V1LocationType } from 'services/api-ts-sdk';
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
  [V1LocationType.TRAINING]: 'Metrics',
  [V1LocationType.HYPERPARAMETERS]: 'Hyperparameters',
} as const;

interface ColumnMenuProps {
  initialVisibleColumns: string[];
  projectColumns: Loadable<ProjectColumn[]>;
  setVisibleColumns: (newColumns: string[]) => void;
}

interface ColumnTabProps {
  columnState: Set<string>;
  handleShowSuggested: () => void;
  searchString: string;
  setSearchString: React.Dispatch<React.SetStateAction<string>>;
  setVisibleColumns: (newColumns: string[]) => void;
  tab: V1LocationType | V1LocationType[];
  totalColumns: ProjectColumn[];
}

const ColumnPickerTab: React.FC<ColumnTabProps> = ({
  columnState,
  handleShowSuggested,
  searchString,
  setSearchString,
  setVisibleColumns,
  tab,
  totalColumns,
}) => {
  const filteredColumns = useMemo(() => {
    const regex = new RegExp(searchString, 'i');
    const locations = Array.isArray(tab) ? tab : [tab];
    return totalColumns.filter(
      (col) => locations.includes(col.location) && regex.test(col.displayName || col.column),
    );
  }, [searchString, totalColumns, tab]);

  const allFilteredColumnsChecked = useMemo(() => {
    return filteredColumns.map((col) => columnState.has(col.column)).every((col) => col === true);
  }, [columnState, filteredColumns]);

  const handleShowHideAll = useCallback(() => {
    const filteredColumnMap: Record<string, boolean> = filteredColumns.reduce(
      (acc, col) => ({ ...acc, [col.column]: columnState.has(col.column) }),
      {},
    );

    const newColumns = allFilteredColumnsChecked
      ? [...columnState].filter((col) => !filteredColumnMap[col])
      : [...new Set([...columnState, ...filteredColumns.map((col) => col.column)])];
    setVisibleColumns(newColumns);
  }, [allFilteredColumnsChecked, filteredColumns, setVisibleColumns, columnState]);

  const handleColumnChange = useCallback(
    (event: CheckboxChangeEvent) => {
      const { id, checked } = event.target;
      if (id === undefined) return;

      const newColumnSet = new Set(columnState);
      checked ? newColumnSet.add(id) : newColumnSet.delete(id);
      setVisibleColumns([...newColumnSet]);
    },
    [columnState, setVisibleColumns],
  );

  const handleSearch = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      setSearchString(e.target.value);
    },
    [setSearchString],
  );

  return (
    <div>
      <Input
        allowClear
        autoFocus
        placeholder="Search"
        value={searchString}
        onChange={handleSearch}
      />
      {totalColumns.length !== 0 ? (
        <Space className={css.columnList} direction="vertical">
          {filteredColumns.length > 0 ? (
            filteredColumns.map((col) => (
              <Checkbox
                checked={columnState.has(col.column)}
                id={col.column}
                key={col.column}
                onChange={handleColumnChange}>
                {col.displayName || col.column}
              </Checkbox>
            ))
          ) : (
            <Empty description="No results" />
          )}
        </Space>
      ) : (
        <Spinner />
      )}
      <div className={css.actionRow}>
        <Button type="text" onClick={handleShowHideAll}>
          {allFilteredColumnsChecked ? 'Hide' : 'Show'} all
        </Button>
        <Button type="text" onClick={handleShowSuggested}>
          Reset
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
  const [searchString, setSearchString] = useState('');
  const [open, setOpen] = useState(false);

  const closeMenu = () => {
    setOpen(false);
  };

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen);
  };

  const totalColumns = useMemo(
    () => removeBannedColumns(Loadable.getOrElse([], projectColumns)),
    [projectColumns],
  );

  const columnState = useMemo(() => new Set(initialVisibleColumns), [initialVisibleColumns]);

  const handleShowSuggested = useCallback(() => {
    setVisibleColumns(defaultExperimentColumns);
    closeMenu();
  }, [setVisibleColumns]);

  return (
    <Popover
      content={
        <div className={css.base}>
          <Pivot
            items={[
              V1LocationType.EXPERIMENT,
              [V1LocationType.VALIDATIONS, V1LocationType.TRAINING],
              V1LocationType.HYPERPARAMETERS,
            ].map((tab) => {
              const canonicalTab = Array.isArray(tab) ? tab[0] : tab;
              return {
                children: (
                  <ColumnPickerTab
                    columnState={columnState}
                    handleShowSuggested={handleShowSuggested}
                    searchString={searchString}
                    setSearchString={setSearchString}
                    setVisibleColumns={setVisibleColumns}
                    tab={tab}
                    totalColumns={totalColumns}
                  />
                ),
                forceRender: true,
                key: canonicalTab,
                label: locationLabelMap[canonicalTab],
              };
            })}
          />
        </div>
      }
      open={open}
      placement="bottom"
      trigger="click"
      onOpenChange={handleOpenChange}>
      <Button icon={<Icon name="columns" title="column picker" />}>Columns</Button>
    </Popover>
  );
};

export default ColumnPickerMenu;
