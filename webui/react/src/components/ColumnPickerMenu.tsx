import Badge from 'hew/Badge';
import Button from 'hew/Button';
import Checkbox, { CheckboxChangeEvent } from 'hew/Checkbox';
import Dropdown from 'hew/Dropdown';
import Icon from 'hew/Icon';
import Input from 'hew/Input';
import Message from 'hew/Message';
import Pivot from 'hew/Pivot';
import Spinner from 'hew/Spinner';
import { Loadable } from 'hew/utils/loadable';
import React, { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { FixedSizeList as List } from 'react-window';

import { V1LocationType } from 'services/api-ts-sdk';
import { ProjectColumn } from 'types';
import { ensureArray } from 'utils/data';

import css from './ColumnPickerMenu.module.scss';

const BANNED_COLUMNS: Set<string> = new Set([]);

const removeBannedColumns = (columns: ProjectColumn[]) =>
  columns.filter((col) => !BANNED_COLUMNS.has(col.column));

export const LOCATION_LABEL_MAP: Record<V1LocationType, string> = {
  [V1LocationType.EXPERIMENT]: 'General',
  [V1LocationType.RUN]: 'General',
  [V1LocationType.VALIDATIONS]: 'Metrics',
  [V1LocationType.TRAINING]: 'Metrics',
  [V1LocationType.CUSTOMMETRIC]: 'Metrics',
  [V1LocationType.HYPERPARAMETERS]: 'Hyperparameters',
  [V1LocationType.RUNHYPERPARAMETERS]: 'Hyperparameters',
  [V1LocationType.RUNMETADATA]: 'Metadata',
  [V1LocationType.UNSPECIFIED]: 'Unspecified',
} as const;

export const COLUMNS_MENU_BUTTON = 'columns-menu-button';

interface ColumnMenuProps {
  isMobile?: boolean;
  initialVisibleColumns: string[];
  defaultVisibleColumns: string[];
  defaultPinnedCount: number;
  onVisibleColumnChange?: (newColumns: string[], pinnedCount?: number) => void;
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
  onVisibleColumnChange?: (newColumns: string[], pinnedCount?: number) => void;
  projectId: number;
  searchString: string;
  setSearchString: React.Dispatch<React.SetStateAction<string>>;
  tab: V1LocationType | V1LocationType[];
  totalColumns: ProjectColumn[];
  compare: boolean;
  pinnedColumnsCount: number;
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
  onHeatmapSelectionRemove,
}) => {
  const checkedColumns = useMemo(
    () => (compare ? new Set(columnState.slice(0, pinnedColumnsCount)) : new Set(columnState)),
    [columnState, compare, pinnedColumnsCount],
  );

  const filteredColumns = useMemo(() => {
    const regex = new RegExp(searchString, 'i');
    const locations = ensureArray(tab);
    return totalColumns
      .filter(
        (col) => locations.includes(col.location) && regex.test(col.displayName || col.column),
      )
      .sort(
        (a, b) =>
          locations.findIndex((l) => l === a.location) -
          locations.findIndex((l) => l === b.location),
      );
  }, [searchString, totalColumns, tab]);

  const allFilteredColumnsChecked = useMemo(() => {
    return filteredColumns.every((col) => columnState.includes(col.column));
  }, [columnState, filteredColumns]);
  const [metadataColumns, setMetadataColumns] = useState(() => new Map<string, number[]>()); // a map of metadata columns and found indexes

  const handleShowHideAll = useCallback(() => {
    const filteredColumnMap: Record<string, boolean> = filteredColumns.reduce(
      (acc, col) => ({ ...acc, [col.column]: columnState.includes(col.column) }),
      {},
    );

    const newColumns = allFilteredColumnsChecked
      ? columnState.filter((col) => !filteredColumnMap[col])
      : [...new Set([...columnState, ...filteredColumns.map((col) => col.column)])];
    const pinnedCount = allFilteredColumnsChecked
      ? // If uncheck something pinned, reduce the pinnedColumnsCount
        newColumns.filter((col) => columnState.indexOf(col) < pinnedColumnsCount).length
      : pinnedColumnsCount;

    onVisibleColumnChange?.(newColumns, pinnedCount);
  }, [
    allFilteredColumnsChecked,
    columnState,
    filteredColumns,
    onVisibleColumnChange,
    pinnedColumnsCount,
  ]);

  const handleColumnChange = useCallback(
    (event: CheckboxChangeEvent) => {
      const { id, checked } = event.target;
      if (id === undefined) return;
      if (compare) {
        // pin or unpin column
        const newColumns = columnState.filter((c) => c !== id);
        let pinnedCount = pinnedColumnsCount;
        if (checked) {
          newColumns.splice(pinnedColumnsCount, 0, id);
          pinnedCount = Math.max(pinnedColumnsCount + 1, 0);
        } else {
          newColumns.splice(pinnedColumnsCount - 1, 0, id);
          pinnedCount = Math.max(pinnedColumnsCount - 1, 0);
        }
        onVisibleColumnChange?.(newColumns, pinnedCount);
      } else {
        let pinnedCount = pinnedColumnsCount;
        // If uncheck something pinned, reduce the pinnedColumnsCount
        if (!checked && columnState.indexOf(id) < pinnedColumnsCount) {
          pinnedCount = Math.max(pinnedColumnsCount - 1, 0);
        }
        // If uncheck something had heatmap skipped, reset to heatmap visible
        if (!checked) {
          onHeatmapSelectionRemove?.(id);
        }
        const newColumnSet = new Set(columnState);
        checked ? newColumnSet.add(id) : newColumnSet.delete(id);
        onVisibleColumnChange?.([...newColumnSet], pinnedCount);
      }
    },
    [compare, columnState, onVisibleColumnChange, onHeatmapSelectionRemove, pinnedColumnsCount],
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
      const getColDisplayName = (col: ProjectColumn) => {
        const metCol = metadataColumns.get(col.column);
        if (metCol !== undefined && metCol.length > 1) {
          return (
            <>
              {col.column} <Badge text={col.type.replace('COLUMN_TYPE_', '').toLowerCase()} />
            </>
          );
        }

        return col.displayName || col.column;
      };
      return (
        <div
          className={css.rows}
          data-test="row"
          data-test-id={col.column}
          key={col.column}
          style={style}>
          <Checkbox
            checked={checkedColumns.has(col.column)}
            data-test="checkbox"
            id={col.column}
            onChange={handleColumnChange}>
            {getColDisplayName(col)}
          </Checkbox>
        </div>
      );
    },
    [filteredColumns, checkedColumns, metadataColumns, handleColumnChange],
  );

  useEffect(() => {
    for (const [index, { column }] of totalColumns.entries()) {
      if (column.includes('metadata')) {
        const columnEntry = metadataColumns.get(column) ?? [];
        if (!columnEntry.includes(index)) {
          setMetadataColumns((prev) => {
            prev.set(column, [...columnEntry, index]);
            return prev;
          });
        }
      }
    }
  }, [totalColumns, metadataColumns]);

  return (
    <div data-test-component="columnPickerTab" data-testid="column-picker-tab">
      <Input
        allowClear
        autoFocus
        data-test="search"
        placeholder="Search"
        value={searchString}
        onChange={handleSearch}
      />
      {totalColumns.length !== 0 ? (
        <div className={css.columns} data-test="columns">
          {filteredColumns.length > 0 ? (
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
          <Button data-test="showAll" type="text" onClick={handleShowHideAll}>
            {allFilteredColumnsChecked ? 'Hide' : 'Show'} all
          </Button>
          <Button data-test="reset" type="text" onClick={handleShowSuggested}>
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
  defaultVisibleColumns,
  defaultPinnedCount,
  projectId,
  isMobile = false,
  onVisibleColumnChange,
  onHeatmapSelectionRemove,
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
    onVisibleColumnChange?.(defaultVisibleColumns, defaultPinnedCount);
    closeMenu();
  }, [onVisibleColumnChange, defaultVisibleColumns, defaultPinnedCount]);

  return (
    <Dropdown
      content={
        <div className={css.base}>
          {tabs.length > 1 && (
            <Pivot
              items={tabs.map((tab) => {
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
                      onHeatmapSelectionRemove={onHeatmapSelectionRemove}
                      onVisibleColumnChange={onVisibleColumnChange}
                    />
                  ),
                  forceRender: true,
                  key: canonicalTab,
                  label: LOCATION_LABEL_MAP[canonicalTab],
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
      <Button
        data-test-component="columnPickerMenu"
        data-testid={COLUMNS_MENU_BUTTON}
        hideChildren={isMobile}
        icon={<Icon name="columns" title="column picker" />}>
        Columns
      </Button>
    </Dropdown>
  );
};

export default ColumnPickerMenu;
