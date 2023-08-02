import { Popover, Space } from 'antd';
import { CheckboxChangeEvent } from 'antd/es/checkbox';
import React, { ChangeEvent, useCallback, useMemo, useState } from 'react';

import Button from 'components/kit/Button';
import Checkbox from 'components/kit/Checkbox';
import Empty from 'components/kit/Empty';
import Icon from 'components/kit/Icon';
import Input from 'components/kit/Input';
import Pivot from 'components/kit/Pivot';
import Spinner from 'components/kit/Spinner';
import { useSettings } from 'hooks/useSettings';
import { V1LocationType } from 'services/api-ts-sdk';
import { ProjectColumn } from 'types';
import { ensureArray } from 'utils/data';
import { Loadable } from 'utils/loadable';

import { F_ExperimentListSettings, settingsConfigForProject } from '../F_ExperimentList.settings';

import css from './ColumnPickerMenu.module.scss';
import { defaultExperimentColumns } from './columns';

const BANNED_COLUMNS: Set<string> = new Set([]);

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
  projectId: number;
  isMobile?: boolean;
}

interface ColumnTabProps {
  columnState: string[];
  handleShowSuggested: () => void;
  searchString: string;
  setSearchString: React.Dispatch<React.SetStateAction<string>>;
  setVisibleColumns: (newColumns: string[]) => void;
  tab: V1LocationType | V1LocationType[];
  totalColumns: ProjectColumn[];
  projectId: number;
}

const ColumnPickerTab: React.FC<ColumnTabProps> = ({
  columnState,
  handleShowSuggested,
  searchString,
  setSearchString,
  setVisibleColumns,
  tab,
  totalColumns,
  projectId,
}) => {
  const settingsConfig = useMemo(() => settingsConfigForProject(projectId), [projectId]);

  const { settings, updateSettings } = useSettings<F_ExperimentListSettings>(settingsConfig);

  const checkedColumn = useMemo(
    () =>
      settings.compare
        ? new Set(columnState.slice(0, settings.pinnedColumnsCount))
        : new Set(columnState),
    [columnState, settings.compare, settings.pinnedColumnsCount],
  );

  const filteredColumns = useMemo(() => {
    const regex = new RegExp(searchString, 'i');
    const locations = ensureArray(tab);
    return totalColumns.filter(
      (col) => locations.includes(col.location) && regex.test(col.displayName || col.column),
    );
  }, [searchString, totalColumns, tab]);

  const allFilteredColumnsChecked = useMemo(() => {
    return filteredColumns.every((col) => columnState.includes(col.column));
  }, [columnState, filteredColumns]);

  const handleShowHideAll = useCallback(() => {
    const filteredColumnMap: Record<string, boolean> = filteredColumns.reduce(
      (acc, col) => ({ ...acc, [col.column]: columnState.includes(col.column) }),
      {},
    );

    const newColumns = allFilteredColumnsChecked
      ? columnState.filter((col) => !filteredColumnMap[col])
      : [...new Set([...columnState, ...filteredColumns.map((col) => col.column)])];
    setVisibleColumns(newColumns);

    // If uncheck something pinned, reduce the pinnedColumnsCount
    allFilteredColumnsChecked &&
      updateSettings({
        pinnedColumnsCount: newColumns.filter(
          (col) => columnState.indexOf(col) < settings.pinnedColumnsCount,
        ).length,
      });
  }, [
    allFilteredColumnsChecked,
    filteredColumns,
    setVisibleColumns,
    columnState,
    updateSettings,
    settings.pinnedColumnsCount,
  ]);

  const handleColumnChange = useCallback(
    (event: CheckboxChangeEvent) => {
      const { id, checked } = event.target;
      if (id === undefined) return;
      const pinnedColumnsCount = settings.pinnedColumnsCount;
      if (settings.compare) {
        // pin or unpin column
        const newColumns = columnState.filter((c) => c !== id);
        if (checked) {
          newColumns.splice(pinnedColumnsCount, 0, id);
          updateSettings({ pinnedColumnsCount: Math.max(pinnedColumnsCount + 1, 0) });
        } else {
          newColumns.splice(pinnedColumnsCount - 1, 0, id);
          updateSettings({ pinnedColumnsCount: Math.max(pinnedColumnsCount - 1, 0) });
        }
        setVisibleColumns(newColumns);
      } else {
        // If uncheck something pinned, reduce the pinnedColumnsCount
        if (!checked && columnState.indexOf(id) < pinnedColumnsCount) {
          updateSettings({ pinnedColumnsCount: Math.max(pinnedColumnsCount - 1, 0) });
        }
        const newColumnSet = new Set(columnState);
        checked ? newColumnSet.add(id) : newColumnSet.delete(id);
        setVisibleColumns([...newColumnSet]);
      }
    },
    [columnState, setVisibleColumns, settings.compare, settings.pinnedColumnsCount, updateSettings],
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
                checked={checkedColumn.has(col.column)}
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
        <Spinner spinning />
      )}
      {!settings.compare && (
        <div className={css.actionRow}>
          <Button type="text" onClick={handleShowHideAll}>
            {allFilteredColumnsChecked ? 'Hide' : 'Show'} all
          </Button>
          <Button type="text" onClick={handleShowSuggested}>
            Reset
          </Button>
        </div>
      )}
    </div>
  );
};

const ColumnPickerMenu: React.FC<ColumnMenuProps> = ({
  projectColumns,
  setVisibleColumns,
  initialVisibleColumns,
  projectId,
  isMobile = false,
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
                    columnState={initialVisibleColumns}
                    handleShowSuggested={handleShowSuggested}
                    projectId={projectId}
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
      <Button hideChildren={isMobile} icon={<Icon name="columns" title="column picker" />}>
        Columns
      </Button>
    </Popover>
  );
};

export default ColumnPickerMenu;
