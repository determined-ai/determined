import { CompactSelection, GridSelection } from '@glideapps/glide-data-grid';
import { isLeft } from 'fp-ts/lib/Either';
import Column from 'hew/Column';
import {
  ColumnDef,
  DEFAULT_COLUMN_WIDTH,
  defaultDateColumn,
  defaultNumberColumn,
  defaultSelectionColumn,
  defaultTextColumn,
  MULTISELECT,
} from 'hew/DataGrid/columns';
import DataGrid, {
  DataGridHandle,
  HandleSelectionChangeType,
  RangelessSelectionType,
  SelectionType,
  Sort,
  validSort,
  ValidSort,
} from 'hew/DataGrid/DataGrid';
import { MenuItem } from 'hew/Dropdown';
import Icon from 'hew/Icon';
import Link from 'hew/Link';
import Message from 'hew/Message';
import Pagination from 'hew/Pagination';
import Row from 'hew/Row';
import { useToast } from 'hew/Toast';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { isUndefined } from 'lodash';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { v4 as uuidv4 } from 'uuid';

import { Error } from 'components/exceptions';
import ExperimentActionDropdown from 'components/ExperimentActionDropdown';
import { FilterFormStore, ROOT_ID } from 'components/FilterForm/components/FilterFormStore';
import {
  AvailableOperators,
  FilterFormSet,
  FormField,
  FormGroup,
  FormKind,
  IOFilterFormSet,
  Operator,
  SpecialColumnNames,
} from 'components/FilterForm/components/type';
import { EMPTY_SORT, sortMenuItemsForColumn } from 'components/MultiSortMenu';
import { RowHeight } from 'components/OptionsMenu';
import { DataGridGlobalSettings, settingsConfigGlobal } from 'components/OptionsMenu.settings';
import TableActionBar from 'components/TableActionBar';
import useUI from 'components/ThemeProvider';
import { useAsync } from 'hooks/useAsync';
import { useGlasbey } from 'hooks/useGlasbey';
import useMobile from 'hooks/useMobile';
import usePolling from 'hooks/usePolling';
import { useSettings } from 'hooks/useSettings';
import { useTypedParams } from 'hooks/useTypedParams';
import { paths } from 'routes/utils';
import { getProjectColumns, searchExperiments } from 'services/api';
import {
  V1BulkExperimentFilters,
  V1ColumnType,
  V1LocationType,
  V1TableType,
} from 'services/api-ts-sdk';
import usersStore from 'stores/users';
import userSettings from 'stores/userSettings';
import {
  BulkExperimentItem,
  ExperimentAction,
  ExperimentWithTrial,
  Project,
  ProjectColumn,
  RunState,
  SelectionType as SelectionState,
} from 'types';
import handleError from 'utils/error';
import { getProjectExperimentForExperimentItem } from 'utils/experiment';
import { eagerSubscribe } from 'utils/observable';
import { pluralizer } from 'utils/string';

import { getColumnDefs, searcherMetricsValColumn } from './columns';
import css from './Searches.module.scss';
import {
  DEFAULT_SELECTION,
  defaultProjectSettings,
  ProjectSettings,
  ProjectUrlSettings,
  settingsPathForProject,
} from './Searches.settings';

interface Props {
  project: Project;
}

type ExperimentWithIndex = { index: number; experiment: BulkExperimentItem };

const BANNED_FILTER_COLUMNS = new Set(['searcherMetricsVal']);
const BANNED_SORT_COLUMNS = new Set(['tags', 'searcherMetricsVal']);

const makeSortString = (sorts: ValidSort[]): string =>
  sorts.map((s) => `${s.column}=${s.direction}`).join(',');

const parseSortString = (sortString: string): Sort[] => {
  if (!sortString) return [EMPTY_SORT];
  const components = sortString.split(',');
  return components.map((c) => {
    const [column, direction] = c.split('=', 2);
    return {
      column,
      direction: direction === 'asc' || direction === 'desc' ? direction : undefined,
    };
  });
};

const formStore = new FilterFormStore();

export const PAGE_SIZE = 100;
const INITIAL_LOADING_EXPERIMENTS: Loadable<ExperimentWithTrial>[] = new Array(PAGE_SIZE).fill(
  NotLoaded,
);

const STATIC_COLUMNS = [MULTISELECT];

const rowHeightMap: Record<RowHeight, number> = {
  [RowHeight.EXTRA_TALL]: 44,
  [RowHeight.TALL]: 40,
  [RowHeight.MEDIUM]: 36,
  [RowHeight.SHORT]: 32,
};

const Searches: React.FC<Props> = ({ project }) => {
  const dataGridRef = useRef<DataGridHandle>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  const settingsPath = useMemo(() => settingsPathForProject(project.id), [project.id]);
  const projectSettingsObs = useMemo(
    () => userSettings.get(ProjectSettings, settingsPath),
    [settingsPath],
  );
  const projectSettings = useObservable(projectSettingsObs);
  const isLoadingSettings = useMemo(() => projectSettings.isNotLoaded, [projectSettings]);
  const updateSettings = useCallback(
    (p: Partial<ProjectSettings>) => userSettings.setPartial(ProjectSettings, settingsPath, p),
    [settingsPath],
  );
  const settings = useMemo(
    () =>
      projectSettings
        .map((s) => ({ ...defaultProjectSettings, ...s }))
        .getOrElse(defaultProjectSettings),
    [projectSettings],
  );

  const { params, updateParams } = useTypedParams(ProjectUrlSettings, {});
  const page = params.page || 0;
  const setPage = useCallback(
    (p: number) => updateParams({ page: p || undefined }),
    [updateParams],
  );

  const { settings: globalSettings, updateSettings: updateGlobalSettings } =
    useSettings<DataGridGlobalSettings>(settingsConfigGlobal);

  const [sorts, setSorts] = useState<Sort[]>(() => {
    if (!isLoadingSettings) {
      return parseSortString(settings.sortString);
    }
    return [EMPTY_SORT];
  });
  const sortString = useMemo(() => makeSortString(sorts.filter(validSort.is)), [sorts]);
  const [experiments, setExperiments] = useState<Loadable<ExperimentWithTrial>[]>(
    INITIAL_LOADING_EXPERIMENTS,
  );
  const [total, setTotal] = useState<Loadable<number>>(NotLoaded);
  const [isOpenFilter, setIsOpenFilter] = useState<boolean>(false);
  const filtersString = useObservable(formStore.asJsonString);
  const loadableFormset = useObservable(formStore.formset);
  const rootFilterChildren: Array<FormGroup | FormField> = Loadable.match(loadableFormset, {
    _: () => [],
    Loaded: (formset: FilterFormSet) => formset.filterGroup.children,
  });
  const isMobile = useMobile();
  const { openToast } = useToast();

  const handlePinnedColumnsCountChange = useCallback(
    (newCount: number) => updateSettings({ pinnedColumnsCount: newCount }),
    [updateSettings],
  );
  const handleIsOpenFilterChange = useCallback((newOpen: boolean) => {
    setIsOpenFilter(newOpen);
    if (!newOpen) {
      formStore.sweep();
    }
  }, []);

  const resetPagination = useCallback(() => {
    setIsLoading(true);
    setPage(0);
    setExperiments(INITIAL_LOADING_EXPERIMENTS);
  }, [setPage]);

  useEffect(() => {
    let cleanup: () => void;
    // eagerSubscribe is like subscribe but it runs once before the observed value changes.
    cleanup = eagerSubscribe(projectSettingsObs, (ps, prevPs) => {
      // init formset once from settings when loaded, then flip the sync
      // direction -- when formset changes, update settings
      if (!prevPs?.isLoaded) {
        ps.forEach((s) => {
          cleanup?.();
          if (!s?.filterset) {
            formStore.init();
          } else {
            const formSetValidation = IOFilterFormSet.decode(JSON.parse(s.filterset));
            if (isLeft(formSetValidation)) {
              handleError(formSetValidation.left, {
                publicSubject: 'Unable to initialize filterset from settings',
              });
            } else {
              formStore.init(formSetValidation.right);
            }
          }
          cleanup = formStore.asJsonString.subscribe(() => {
            resetPagination();
            const loadableFormset = formStore.formset.get();
            Loadable.forEach(loadableFormset, (formSet) =>
              updateSettings({
                filterset: JSON.stringify(formSet),
                selection: DEFAULT_SELECTION,
              }),
            );
          });
        });
      }
    });
    return () => cleanup?.();
  }, [projectSettingsObs, resetPagination, updateSettings]);

  const [isLoading, setIsLoading] = useState(true);
  const [error] = useState(false);
  const [canceler] = useState(new AbortController());

  const allSelectedExperimentIds = useMemo(() => {
    if (settings.selection.type === 'ONLY_IN') {
      return settings.selection.selections;
    }
    return [];
  }, [settings.selection]);

  const loadedSelectedExperimentIds = useMemo(() => {
    const selectedMap = new Map<number, ExperimentWithIndex>();
    if (isLoadingSettings) {
      return selectedMap;
    }
    const selectedIdSet = new Set(allSelectedExperimentIds);
    experiments.forEach((e, index) => {
      Loadable.forEach(e, ({ experiment }) => {
        if (selectedIdSet.has(experiment.id)) {
          selectedMap.set(experiment.id, { experiment, index });
        }
      });
    });
    return selectedMap;
  }, [isLoadingSettings, allSelectedExperimentIds, experiments]);

  const selection = useMemo<GridSelection>(() => {
    let rows = CompactSelection.empty();
    loadedSelectedExperimentIds.forEach((info) => {
      rows = rows.add(info.index);
    });
    return {
      columns: CompactSelection.empty(),
      rows,
    };
  }, [loadedSelectedExperimentIds]);

  const colorMap = useGlasbey([...loadedSelectedExperimentIds.keys()]);

  const experimentFilters = useMemo(() => {
    const filters: V1BulkExperimentFilters = {
      projectId: project.id,
    };
    return filters;
  }, [project.id]);

  const numFilters = useMemo(() => {
    return (
      Object.values(experimentFilters).filter((x) => x !== undefined).length -
      1 +
      rootFilterChildren.length
    );
  }, [experimentFilters, rootFilterChildren.length]);

  const handleSortChange = useCallback(
    (sorts: Sort[]) => {
      setSorts(sorts);
      const newSortString = makeSortString(sorts.filter(validSort.is));
      if (newSortString !== sortString) {
        resetPagination();
      }
      updateSettings({ sortString: newSortString });
    },
    [resetPagination, sortString, updateSettings],
  );

  useEffect(() => {
    if (!isLoadingSettings && settings.sortString) {
      setSorts(parseSortString(settings.sortString));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoadingSettings]);

  const fetchExperiments = useCallback(async (): Promise<void> => {
    if (isLoadingSettings || Loadable.isNotLoaded(loadableFormset)) return;
    try {
      // always filter out single trial experiments
      const filters = JSON.parse(filtersString);
      const existingFilterGroup = { ...filters.filterGroup };
      const singleTrialFilter = {
        columnName: 'searcherType',
        kind: 'field',
        location: 'LOCATION_TYPE_EXPERIMENT',
        operator: '!=',
        type: 'COLUMN_TYPE_TEXT',
        value: 'single',
      };
      filters.filterGroup = {
        children: [existingFilterGroup, singleTrialFilter],
        conjunction: 'and',
        kind: 'group',
      };

      const response = await searchExperiments(
        {
          ...experimentFilters,
          filter: JSON.stringify(filters),
          limit: settings.pageLimit,
          offset: page * settings.pageLimit,
          sort: sortString || undefined,
        },
        { signal: canceler.signal },
      );
      const loadedExperiments = response.experiments;

      setExperiments(loadedExperiments.map((experiment) => Loaded(experiment)));
      setTotal(
        response.pagination.total !== undefined ? Loaded(response.pagination.total) : NotLoaded,
      );
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch experiments.' });
    } finally {
      setIsLoading(false);
    }
  }, [
    canceler.signal,
    experimentFilters,
    filtersString,
    isLoadingSettings,
    loadableFormset,
    page,
    sortString,
    settings.pageLimit,
  ]);

  const { stopPolling } = usePolling(fetchExperiments, { rerunOnNewFn: true });

  const projectColumns = useAsync(async () => {
    try {
      const columns = await getProjectColumns({
        id: project.id,
        tableType: V1TableType.EXPERIMENT,
      });
      return columns.filter((c) => c.location === V1LocationType.EXPERIMENT);
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch project columns' });
      return NotLoaded;
    }
  }, [project.id]);

  useEffect(() => {
    return () => {
      canceler.abort();
      stopPolling();
    };
  }, [canceler, stopPolling]);

  const rowRangeToIds = useCallback(
    (range: [number, number]) => {
      const slice = experiments.slice(range[0], range[1]);
      return Loadable.filterNotLoaded(slice).map(({ experiment }) => experiment.id);
    },
    [experiments],
  );

  const handleSelectionChange: HandleSelectionChangeType = useCallback(
    (selectionType: SelectionType | RangelessSelectionType, range?: [number, number]) => {
      let newSettings: SelectionState = { ...settings.selection };

      switch (selectionType) {
        case 'add':
          if (!range) return;
          if (newSettings.type === 'ALL_EXCEPT') {
            const excludedSet = new Set(newSettings.exclusions);
            rowRangeToIds(range).forEach((id) => excludedSet.delete(id));
            newSettings.exclusions = Array.from(excludedSet);
          } else {
            const includedSet = new Set(newSettings.selections);
            rowRangeToIds(range).forEach((id) => includedSet.add(id));
            newSettings.selections = Array.from(includedSet);
          }

          break;
        case 'add-all':
          newSettings = {
            exclusions: [],
            type: 'ALL_EXCEPT' as const,
          };

          break;
        case 'remove':
          if (!range) return;
          if (newSettings.type === 'ALL_EXCEPT') {
            const excludedSet = new Set(newSettings.exclusions);
            rowRangeToIds(range).forEach((id) => excludedSet.add(id));
            newSettings.exclusions = Array.from(excludedSet);
          } else {
            const includedSet = new Set(newSettings.selections);
            rowRangeToIds(range).forEach((id) => includedSet.delete(id));
            newSettings.selections = Array.from(includedSet);
          }

          break;
        case 'remove-all':
          newSettings = DEFAULT_SELECTION;

          break;
        case 'set':
          if (!range) return;
          newSettings = {
            ...DEFAULT_SELECTION,
            selections: Array.from(rowRangeToIds(range)),
          };

          break;
      }

      updateSettings({ selection: newSettings });
    },
    [rowRangeToIds, settings.selection, updateSettings],
  );

  const handleActionComplete = useCallback(async () => {
    /**
     * Deselect selected rows since their states may have changed where they
     * are no longer part of the filter criteria.
     */
    handleSelectionChange('remove-all');

    // Re-fetch experiment list to get updates based on batch action.
    await fetchExperiments();
  }, [handleSelectionChange, fetchExperiments]);

  const handleActionSuccess = useCallback(
    (
      action: ExperimentAction,
      successfulIds: number[],
      data?: Partial<BulkExperimentItem>,
    ): void => {
      const idSet = new Set(successfulIds);
      const updateExperiment = (updated: Partial<BulkExperimentItem>) => {
        setExperiments((prev) =>
          prev.map((expLoadable) =>
            Loadable.map(expLoadable, (experiment) =>
              idSet.has(experiment.experiment.id)
                ? { ...experiment, experiment: { ...experiment.experiment, ...updated } }
                : experiment,
            ),
          ),
        );
      };
      switch (action) {
        case ExperimentAction.Activate:
          updateExperiment({ state: RunState.Active });
          break;
        case ExperimentAction.Archive:
          updateExperiment({ archived: true });
          break;
        case ExperimentAction.Cancel:
          updateExperiment({ state: RunState.StoppingCanceled });
          break;
        case ExperimentAction.Kill:
          updateExperiment({ state: RunState.StoppingKilled });
          break;
        case ExperimentAction.Pause:
          updateExperiment({ state: RunState.Paused });
          break;
        case ExperimentAction.Unarchive:
          updateExperiment({ archived: false });
          break;
        case ExperimentAction.Edit:
          if (data) updateExperiment(data);
          openToast({ severity: 'Confirm', title: 'Experiment updated successfully' });
          break;
        case ExperimentAction.Move:
        case ExperimentAction.Delete:
          setExperiments((prev) =>
            prev.filter((expLoadable) =>
              Loadable.match(expLoadable, {
                _: () => true,
                Loaded: (experiment) => !idSet.has(experiment.experiment.id),
              }),
            ),
          );
          break;
        case ExperimentAction.RetainLogs:
          break;
        // Exhaustive cases to ignore.
        default:
          break;
      }
      handleSelectionChange('remove-all');
    },
    [handleSelectionChange, openToast],
  );

  const handleContextMenuComplete = useCallback(
    (action: ExperimentAction, id: number, data?: Partial<BulkExperimentItem>) =>
      handleActionSuccess(action, [id], data),
    [handleActionSuccess],
  );

  const handleColumnsOrderChange = useCallback(
    // changing both column order and pinned count should happen in one update:
    (newColumnsOrder: string[], pinnedCount?: number) => {
      const newColumnWidths = newColumnsOrder
        .filter((c) => !(c in settings.columnWidths))
        .reduce((acc: Record<string, number>, col) => {
          acc[col] = DEFAULT_COLUMN_WIDTH;
          return acc;
        }, {});
      updateSettings({
        columns: newColumnsOrder,
        columnWidths: {
          ...settings.columnWidths,
          ...newColumnWidths,
        },
        pinnedColumnsCount: isUndefined(pinnedCount) ? settings.pinnedColumnsCount : pinnedCount,
      });
    },
    [updateSettings, settings.pinnedColumnsCount, settings.columnWidths],
  );

  const handleRowHeightChange = useCallback(
    (newRowHeight: RowHeight) => {
      updateGlobalSettings({ rowHeight: newRowHeight });
    },
    [updateGlobalSettings],
  );

  useEffect(() => {
    const handleEsc = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        handleSelectionChange('remove-all');
      }
    };
    window.addEventListener('keydown', handleEsc);

    return () => {
      window.removeEventListener('keydown', handleEsc);
    };
  }, [handleSelectionChange]);

  const onPageChange = useCallback(
    (cPage: number, cPageSize: number) => {
      updateSettings({ pageLimit: cPageSize });
      // Pagination component is assuming starting index of 1.
      if (cPage - 1 !== page) {
        setExperiments(Array(cPageSize).fill(NotLoaded));
      }
      setPage(cPage - 1);
    },
    [page, updateSettings, setPage],
  );

  const handleColumnWidthChange = useCallback(
    (columnId: string, width: number) => {
      updateSettings({
        columnWidths: {
          ...settings.columnWidths,
          [columnId]: width,
        },
      });
    },
    [updateSettings, settings.columnWidths],
  );

  const columnsIfLoaded = useMemo(
    () => (isLoadingSettings ? [] : settings.columns),
    [isLoadingSettings, settings.columns],
  );

  const {
    ui: { theme: appTheme },
    isDarkMode,
  } = useUI();

  const users = useObservable(usersStore.getUsers());

  const columns: ColumnDef<ExperimentWithTrial>[] = useMemo(() => {
    const projectColumnsMap: Loadable<Record<string, ProjectColumn>> = Loadable.map(
      projectColumns,
      (columns) => {
        return columns.reduce((acc, col) => ({ ...acc, [col.column]: col }), {});
      },
    );
    const columnDefs = getColumnDefs({
      appTheme,
      columnWidths: settings.columnWidths,
      themeIsDark: isDarkMode,
      users,
    });
    const gridColumns = [...STATIC_COLUMNS, ...columnsIfLoaded]
      .map((columnName) => {
        if (columnName === MULTISELECT) {
          return (columnDefs[columnName] = defaultSelectionColumn(selection.rows, false));
        }

        if (!Loadable.isLoaded(projectColumnsMap)) {
          if (columnName in columnDefs) return columnDefs[columnName];
          return;
        }

        const currentColumn = projectColumnsMap.data[columnName];
        if (!currentColumn) {
          if (columnName in columnDefs) return columnDefs[columnName];
          return;
        }

        // prioritize column title from getProjectColumns API response, but use static front-end definition as fallback:
        if (columnName in columnDefs)
          return {
            ...columnDefs[columnName],
            title: currentColumn.displayName ?? columnDefs[columnName].title,
          };

        let dataPath: string | undefined = undefined;
        switch (currentColumn.location) {
          case V1LocationType.EXPERIMENT:
            dataPath = `experiment.${currentColumn.column}`;
            break;
          case V1LocationType.UNSPECIFIED:
          default:
            break;
        }
        switch (currentColumn.type) {
          case V1ColumnType.NUMBER: {
            columnDefs[currentColumn.column] = defaultNumberColumn(
              currentColumn.column,
              currentColumn.displayName || currentColumn.column,
              settings.columnWidths[currentColumn.column],
              dataPath,
            );
            break;
          }
          case V1ColumnType.DATE:
            columnDefs[currentColumn.column] = defaultDateColumn(
              currentColumn.column,
              currentColumn.displayName || currentColumn.column,
              settings.columnWidths[currentColumn.column],
              dataPath,
            );
            break;
          case V1ColumnType.TEXT:
          case V1ColumnType.UNSPECIFIED:
          default:
            columnDefs[currentColumn.column] = defaultTextColumn(
              currentColumn.column,
              currentColumn.displayName || currentColumn.column,
              settings.columnWidths[currentColumn.column],
              dataPath,
            );
        }
        if (currentColumn.column === 'searcherMetricsVal') {
          columnDefs[currentColumn.column] = searcherMetricsValColumn(
            settings.columnWidths[currentColumn.column],
          );
        }
        return columnDefs[currentColumn.column];
      })
      .flatMap((col) => (col ? [col] : []));
    return gridColumns;
  }, [
    projectColumns,
    settings.columnWidths,
    columnsIfLoaded,
    appTheme,
    isDarkMode,
    selection.rows,
    users,
  ]);

  const getHeaderMenuItems = (columnId: string, colIdx: number): MenuItem[] => {
    if (columnId === MULTISELECT) {
      const items: MenuItem[] = [
        settings.selection.type === 'ALL_EXCEPT' || settings.selection.selections.length > 0
          ? {
              key: 'select-none',
              label: 'Clear selected',
              onClick: () => {
                handleSelectionChange?.('remove-all');
              },
            }
          : null,
        ...[5, 10, 25].map((n) => ({
          key: `select-${n}`,
          label: `Select first ${n}`,
          onClick: () => {
            handleSelectionChange?.('set', [0, n]);
            dataGridRef.current?.scrollToTop();
          },
        })),
        {
          key: 'select-all',
          label: 'Select all',
          onClick: () => {
            handleSelectionChange?.('add', [0, settings.pageLimit]);
          },
        },
      ];
      return items;
    }
    const column = Loadable.getOrElse([], projectColumns).find((c) => c.column === columnId);
    if (!column) {
      return [];
    }

    const filterCount = formStore.getFieldCount(column.column).get();

    const loadableFormset = formStore.formset.get();
    const filterMenuItemsForColumn = () => {
      const isSpecialColumn = (SpecialColumnNames as ReadonlyArray<string>).includes(column.column);
      formStore.addChild(ROOT_ID, FormKind.Field, {
        index: Loadable.match(loadableFormset, {
          _: () => 0,
          Loaded: (formset) => formset.filterGroup.children.length,
        }),
        item: {
          columnName: column.column,
          id: uuidv4(),
          kind: FormKind.Field,
          location: column.location,
          operator: isSpecialColumn ? Operator.Eq : AvailableOperators[column.type][0],
          type: column.type,
          value: null,
        },
      });
      handleIsOpenFilterChange?.(true);
    };
    const clearFilterForColumn = () => {
      formStore.removeByField(column.column);
    };

    const isPinned = colIdx <= settings.pinnedColumnsCount + STATIC_COLUMNS.length - 1;
    const items: MenuItem[] = [
      // Column is pinned if the index is inside of the frozen columns
      colIdx < STATIC_COLUMNS.length || isMobile
        ? null
        : !isPinned
          ? {
              icon: <Icon decorative name="pin" />,
              key: 'pin',
              label: 'Pin column',
              onClick: () => {
                const newColumnsOrder = columnsIfLoaded.filter((c) => c !== column.column);
                newColumnsOrder.splice(settings.pinnedColumnsCount, 0, column.column);
                handleColumnsOrderChange?.(
                  newColumnsOrder,
                  Math.min(settings.pinnedColumnsCount + 1, columnsIfLoaded.length),
                );
              },
            }
          : {
              disabled: settings.pinnedColumnsCount <= 1,
              icon: <Icon decorative name="pin" />,
              key: 'unpin',
              label: 'Unpin column',
              onClick: () => {
                const newColumnsOrder = columnsIfLoaded.filter((c) => c !== column.column);
                newColumnsOrder.splice(settings.pinnedColumnsCount - 1, 0, column.column);
                handleColumnsOrderChange?.(
                  newColumnsOrder,
                  Math.max(settings.pinnedColumnsCount - 1, 0),
                );
              },
            },
      {
        icon: <Icon decorative name="eye-close" />,
        key: 'hide',
        label: 'Hide column',
        onClick: () => {
          const newColumnsOrder = columnsIfLoaded.filter((c) => c !== column.column);
          if (isPinned) {
            handleColumnsOrderChange?.(
              newColumnsOrder,
              Math.max(settings.pinnedColumnsCount - 1, 0),
            );
          } else {
            handleColumnsOrderChange?.(newColumnsOrder);
          }
        },
      },
      ...(BANNED_SORT_COLUMNS.has(column.column)
        ? []
        : [
            { type: 'divider' as const },
            ...sortMenuItemsForColumn(column, sorts, handleSortChange),
          ]),
      ...(BANNED_FILTER_COLUMNS.has(column.column)
        ? []
        : [
            { type: 'divider' as const },
            {
              icon: <Icon decorative name="filter" />,
              key: 'filter',
              label: 'Add Filter',
              onClick: () => {
                setTimeout(filterMenuItemsForColumn, 5);
              },
            },
          ]),
      filterCount > 0
        ? {
            icon: <Icon decorative name="filter" />,
            key: 'filter-clear',
            label: `Clear ${pluralizer(filterCount, 'Filter')}  (${filterCount})`,
            onClick: () => {
              setTimeout(clearFilterForColumn, 5);
            },
          }
        : null,
    ];
    return items;
  };

  const getRowAccentColor = (rowData: ExperimentWithTrial) => {
    return colorMap[rowData.experiment.id];
  };

  return (
    <>
      <TableActionBar
        bannedFilterColumns={BANNED_FILTER_COLUMNS}
        bannedSortColumns={BANNED_SORT_COLUMNS}
        columnGroups={[V1LocationType.EXPERIMENT]}
        formStore={formStore}
        initialVisibleColumns={columnsIfLoaded}
        isOpenFilter={isOpenFilter}
        labelPlural="searches"
        labelSingular="search"
        project={project}
        projectColumns={projectColumns}
        rowHeight={globalSettings.rowHeight}
        selectedExperimentIds={allSelectedExperimentIds}
        sorts={sorts}
        total={total}
        onActionComplete={handleActionComplete}
        onActionSuccess={handleActionSuccess}
        onIsOpenFilterChange={handleIsOpenFilterChange}
        onRowHeightChange={handleRowHeightChange}
        onSortChange={handleSortChange}
        onVisibleColumnChange={handleColumnsOrderChange}
      />
      <div className={css.content} ref={contentRef}>
        {!isLoading && experiments.length === 0 ? (
          numFilters === 0 ? (
            <Message
              action={
                <Link external href={paths.docs('/get-started/webui-qs.html')}>
                  Quick Start Guide
                </Link>
              }
              description="Keep track of searches in a project by connecting up your code."
              icon="experiment"
              title="No Searches"
            />
          ) : (
            <Message description="No results matching your filters" icon="search" />
          )
        ) : error ? (
          <Error />
        ) : (
          <div className={css.paneWrapper}>
            <DataGrid<ExperimentWithTrial, ExperimentAction, BulkExperimentItem>
              columns={columns}
              data={experiments}
              getHeaderMenuItems={getHeaderMenuItems}
              getRowAccentColor={getRowAccentColor}
              imperativeRef={dataGridRef}
              pinnedColumnsCount={isLoadingSettings ? 0 : settings.pinnedColumnsCount}
              renderContextMenuComponent={({
                cell,
                rowData,
                link,
                open,
                onComplete,
                onClose,
                onVisibleChange,
              }) => {
                return (
                  <ExperimentActionDropdown
                    cell={cell}
                    experiment={getProjectExperimentForExperimentItem(rowData.experiment, project)}
                    link={link}
                    makeOpen={open}
                    onComplete={onComplete}
                    onLink={onClose}
                    onVisibleChange={onVisibleChange}>
                    <div />
                  </ExperimentActionDropdown>
                );
              }}
              rowHeight={rowHeightMap[globalSettings.rowHeight as RowHeight]}
              selection={selection}
              sorts={sorts}
              staticColumns={STATIC_COLUMNS}
              onColumnResize={handleColumnWidthChange}
              onColumnsOrderChange={handleColumnsOrderChange}
              onContextMenuComplete={handleContextMenuComplete}
              onPinnedColumnsCountChange={handlePinnedColumnsCountChange}
              onSelectionChange={handleSelectionChange}
            />
            {!isMobile && (
              <Row>
                <Column align="right">
                  <Pagination
                    current={page + 1}
                    pageSize={settings.pageLimit}
                    pageSizeOptions={[20, 40, 80]}
                    total={Loadable.getOrElse(0, total)}
                    onChange={onPageChange}
                  />
                </Column>
              </Row>
            )}
          </div>
        )}
      </div>
    </>
  );
};

export default Searches;
