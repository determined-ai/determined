import Button from 'hew/Button';
import Checkbox, { CheckboxChangeEvent } from 'hew/Checkbox';
import Dropdown from 'hew/Dropdown';
import Icon from 'hew/Icon';
import Input from 'hew/Input';
import Message from 'hew/Message';
import Pivot from 'hew/Pivot';
import Spinner from 'hew/Spinner';
import { Loadable } from 'hew/utils/loadable';
import React, { ChangeEvent, useCallback, useMemo, useState } from 'react';
import { FixedSizeList as List } from 'react-window';

import { defaultExperimentColumns } from 'pages/F_ExpList/expListColumns';
import { V1LocationType } from 'services/api-ts-sdk';
import { ProjectColumn } from 'types';
import { ensureArray } from 'utils/data';

import css from './ColumnPickerMenu.module.scss';

export const COLUMN_PICKER_MENU = 'column-picker-menu';

const BANNED_COLUMNS: Set<string> = new Set([]);

const removeBannedColumns = (columns: ProjectColumn[]) =>
  columns?.filter((col) => !BANNED_COLUMNS.has(col.column));

const locationLabelMap = {
  [V1LocationType.EXPERIMENT]: 'General',
  [V1LocationType.VALIDATIONS]: 'Metrics',
  [V1LocationType.TRAINING]: 'Metrics',
  [V1LocationType.CUSTOMMETRIC]: 'Metrics',
  [V1LocationType.HYPERPARAMETERS]: 'Hyperparameters',
} as const;

interface ColumnMenuProps {
  isMobile?: boolean;
  initialVisibleColumns: string[];
  onVisibleColumnChange?: (newColumns: string[]) => void;
  onPinnedColumnsCountChange?: (newCount: number) => void;
  onHeatmapSelectionRemove?: (id: string) => void;
  projectColumns: Loadable<ProjectColumn[]>;
  projectId: number;
  tabs: (V1LocationType | V1LocationType[])[];
  compare?: boolean;
  pinnedColumnsCount: number;
}

interface ColumnTabProps {
  columnState: string[];
  handleShowSuggested: () => void;
  onVisibleColumnChange?: (newColumns: string[]) => void;
  projectId: number;
  searchString: string;
  setSearchString: React.Dispatch<React.SetStateAction<string>>;
  tab: V1LocationType | V1LocationType[];
  totalColumns: ProjectColumn[];
  compare: boolean;
  pinnedColumnsCount: number;
  onPinnedColumnsCountChange?: (newCount: number) => void;
  onHeatmapSelectionRemove?: (id: string) => void;
}

const ColumnPickerTab: React.FC<ColumnTabProps> = ({
  columnState,
  compare,
  pinnedColumnsCount,
  handleShowSuggested,
  searchString,
  setSearchString,
  tab,
  totalColumns,
  onVisibleColumnChange,
  onPinnedColumnsCountChange,
  onHeatmapSelectionRemove,
}) => {
  const checkedColumns = useMemo(
    () =>
      compare
        ? new Set(columnState.slice(0, pinnedColumnsCount))
        : new Set(columnState),
    [columnState, compare, pinnedColumnsCount],
  );

  const filteredColumns = useMemo(() => {
    const regex = new RegExp(searchString, 'i');
    const locations = ensureArray(tab);
    return totalColumns?.filter(
      (col) => locations.includes(col.location) && regex.test(col.displayName || col.column),
    );
  }, [searchString, totalColumns, tab]);

  const allFilteredColumnsChecked = useMemo(() => {
    return filteredColumns?.every((col) => columnState.includes(col.column));
  }, [columnState, filteredColumns]);

  const handleShowHideAll = useCallback(() => {
    const filteredColumnMap: Record<string, boolean> = filteredColumns.reduce(
      (acc, col) => ({ ...acc, [col.column]: columnState.includes(col.column) }),
      {},
    );

    const newColumns = allFilteredColumnsChecked
      ? columnState.filter((col) => !filteredColumnMap[col])
      : [...new Set([...columnState, ...filteredColumns.map((col) => col.column)])];
    onVisibleColumnChange?.(newColumns);

    // If uncheck something pinned, reduce the pinnedColumnsCount
    allFilteredColumnsChecked &&
      onPinnedColumnsCountChange?.(newColumns?.filter(
        (col) => columnState.indexOf(col) < pinnedColumnsCount,
      ).length);
  }, [
    allFilteredColumnsChecked,
    columnState,
    filteredColumns,
    onVisibleColumnChange,
    pinnedColumnsCount,
    onPinnedColumnsCountChange,
  ]);

  const handleColumnChange = useCallback(
    (event: CheckboxChangeEvent) => {
      const { id, checked } = event.target;
      if (id === undefined) return;
      if (compare) {
        // pin or unpin column
        const newColumns = columnState.filter((c) => c !== id);
        if (checked) {
          newColumns.splice(pinnedColumnsCount, 0, id);
          onPinnedColumnsCountChange?.(Math.max(pinnedColumnsCount + 1, 0));
        } else {
          newColumns.splice(pinnedColumnsCount - 1, 0, id);
          onPinnedColumnsCountChange?.(Math.max(pinnedColumnsCount - 1, 0));
        }
        onVisibleColumnChange?.(newColumns);
      } else {
        // If uncheck something pinned, reduce the pinnedColumnsCount
        if (!checked && columnState.indexOf(id) < pinnedColumnsCount) {
          onPinnedColumnsCountChange?.(Math.max(pinnedColumnsCount - 1, 0));
        }
        // If uncheck something had heatmap skipped, reset to heatmap visible
        if (!checked) {
          onHeatmapSelectionRemove?.(id);
        }
        const newColumnSet = new Set(columnState);
        checked ? newColumnSet.add(id) : newColumnSet.delete(id);
        onVisibleColumnChange?.([...newColumnSet]);
      }
    },
    [
      compare,
      columnState,
      onVisibleColumnChange,
      onHeatmapSelectionRemove,
      pinnedColumnsCount,
      onPinnedColumnsCountChange,
    ],
  );

  const handleSearch = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      setSearchString(e.target.value);
    },
    [setSearchString],
  );

  const rows = useCallback(
    ({ index, style }: { index: number; style: React.CSSProperties }) => {
      const col = filteredColumns[index];
      return (
        <div className={css.rows} key={col.column} style={style}>
          <Checkbox
            checked={checkedColumns.has(col.column)}
            id={col.column}
            onChange={handleColumnChange}>
            {col.displayName || col.column}
          </Checkbox>
        </div>
      );
    },
    [filteredColumns, checkedColumns, handleColumnChange],
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
      {totalColumns?.length !== 0 ? (
        <div className={css.columns}>
          {filteredColumns?.length > 0 ? (
            <List height={360} itemCount={filteredColumns.length} itemSize={30} width="100%">
              {rows}
            </List>
          ) : (
            <Message description="No results" icon="warning" />
          )}
        </div>
      ) : (
        <Spinner spinning />
      )}
      {!compare && (
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
  compare = false,
  pinnedColumnsCount,
  projectColumns,
  initialVisibleColumns,
  projectId,
  isMobile = false,
  onVisibleColumnChange,
  tabs,
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
    onVisibleColumnChange?.(defaultExperimentColumns);
    closeMenu();
  }, [onVisibleColumnChange]);

  return (
    <Dropdown
      content={
        <div className={css.base} data-testid={COLUMN_PICKER_MENU}>
          {tabs.length > 1 && (
            <Pivot
              items={[
                V1LocationType.EXPERIMENT,
                [V1LocationType.VALIDATIONS, V1LocationType.TRAINING, V1LocationType.CUSTOMMETRIC],
                V1LocationType.HYPERPARAMETERS,
              ].map((tab) => {
                const canonicalTab = Array.isArray(tab) ? tab[0] : tab;
                return {
                  children: (
                    <ColumnPickerTab
                      columnState={initialVisibleColumns}
                      compare={compare}
                      handleShowSuggested={handleShowSuggested}
                      pinnedColumnsCount={pinnedColumnsCount}
                      projectId={projectId}
                      searchString={searchString}
                      setSearchString={setSearchString}
                      tab={tab}
                      totalColumns={totalColumns}
                      onVisibleColumnChange={onVisibleColumnChange}
                    />
                  ),
                  forceRender: true,
                  key: canonicalTab,
                  label: locationLabelMap[canonicalTab],
                };
              })}
            />
          )}
          {tabs.length === 1 && (
            <ColumnPickerTab
              columnState={initialVisibleColumns}
              compare={compare}
              handleShowSuggested={handleShowSuggested}
              pinnedColumnsCount={pinnedColumnsCount}
              projectId={projectId}
              searchString={searchString}
              setSearchString={setSearchString}
              tab={tabs[0]}
              totalColumns={totalColumns}
              onVisibleColumnChange={onVisibleColumnChange}
            />
          )}
        </div>
      }
      open={open}
      onOpenChange={handleOpenChange}>
      <Button hideChildren={isMobile} icon={<Icon name="columns" title="column picker" />}>
        Columns
      </Button>
    </Dropdown>
  );
};

export default ColumnPickerMenu;
