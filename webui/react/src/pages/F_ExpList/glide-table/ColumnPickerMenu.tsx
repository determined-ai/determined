import Button from 'hew/Button';
import Checkbox, { CheckboxChangeEvent } from 'hew/Checkbox';
import Dropdown from 'hew/Dropdown';
import Icon from 'hew/Icon';
import Input from 'hew/Input';
import Message from 'hew/Message';
import Pivot from 'hew/Pivot';
import Spinner from 'hew/Spinner';
import { Loadable } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';
import React, { ChangeEvent, useCallback, useMemo, useState } from 'react';
import { FixedSizeList as List } from 'react-window';

import {
  defaultProjectSettings,
  F_ExperimentListSettings,
  ProjectSettings,
  settingsPathForProject,
} from 'pages/F_ExpList/F_ExperimentList.settings';
import { V1LocationType } from 'services/api-ts-sdk';
import userSettings from 'stores/userSettings';
import { ProjectColumn } from 'types';
import { ensureArray } from 'utils/data';

import css from './ColumnPickerMenu.module.scss';
import { defaultExperimentColumns } from './columns';

const BANNED_COLUMNS: Set<string> = new Set([]);

const removeBannedColumns = (columns: ProjectColumn[]) =>
  columns.filter((col) => !BANNED_COLUMNS.has(col.column));

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
  projectColumns: Loadable<ProjectColumn[]>;
  projectId: number;
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
}

const ColumnPickerTab: React.FC<ColumnTabProps> = ({
  columnState,
  handleShowSuggested,
  projectId,
  searchString,
  setSearchString,
  tab,
  totalColumns,
  onVisibleColumnChange,
}) => {
  const settingsPath = useMemo(() => settingsPathForProject(projectId), [projectId]);
  const projectSettings = useObservable(userSettings.get(ProjectSettings, settingsPath));
  const updateSettings = useCallback(
    (p: Partial<F_ExperimentListSettings>) =>
      userSettings.setPartial(ProjectSettings, settingsPath, p),
    [settingsPath],
  );
  const settings = useMemo(
    () =>
      projectSettings
        .map((s) => ({ ...s, ...defaultProjectSettings }))
        .getOrElse(defaultProjectSettings),
    [projectSettings],
  );

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
    onVisibleColumnChange?.(newColumns);

    // If uncheck something pinned, reduce the pinnedColumnsCount
    allFilteredColumnsChecked &&
      updateSettings({
        pinnedColumnsCount: newColumns.filter(
          (col) => columnState.indexOf(col) < settings.pinnedColumnsCount,
        ).length,
      });
  }, [
    allFilteredColumnsChecked,
    columnState,
    filteredColumns,
    onVisibleColumnChange,
    settings.pinnedColumnsCount,
    updateSettings,
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
        onVisibleColumnChange?.(newColumns);
      } else {
        // If uncheck something pinned, reduce the pinnedColumnsCount
        if (!checked && columnState.indexOf(id) < pinnedColumnsCount) {
          updateSettings({ pinnedColumnsCount: Math.max(pinnedColumnsCount - 1, 0) });
        }
        // If uncheck something had heatmap skipped, reset to heatmap visible
        if (!checked) {
          updateSettings({ heatmapSkipped: settings.heatmapSkipped.filter((s) => s !== id) });
        }
        const newColumnSet = new Set(columnState);
        checked ? newColumnSet.add(id) : newColumnSet.delete(id);
        onVisibleColumnChange?.([...newColumnSet]);
      }
    },
    [
      columnState,
      onVisibleColumnChange,
      settings.compare,
      settings.pinnedColumnsCount,
      settings.heatmapSkipped,
      updateSettings,
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
            checked={checkedColumn.has(col.column)}
            id={col.column}
            onChange={handleColumnChange}>
            {col.displayName || col.column}
          </Checkbox>
        </div>
      );
    },
    [filteredColumns, checkedColumn, handleColumnChange],
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
        <div className={css.columns}>
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
  initialVisibleColumns,
  projectId,
  isMobile = false,
  onVisibleColumnChange,
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
        <div className={css.base}>
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
                    handleShowSuggested={handleShowSuggested}
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
