import { CompactSelection, GridSelection } from '@glideapps/glide-data-grid';
import { isLeft } from 'fp-ts/lib/Either';
import Button from 'hew/Button';
import Column from 'hew/Column';
import {
  ColumnDef,
  DEFAULT_COLUMN_WIDTH,
  defaultArrayColumn,
  defaultDateColumn,
  defaultNumberColumn,
  defaultSelectionColumn,
  defaultTextColumn,
  MIN_COLUMN_WIDTH,
  MULTISELECT,
} from 'hew/DataGrid/columns';
import { ContextMenuCompleteHandlerProps } from 'hew/DataGrid/contextMenu';
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
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { v4 as uuidv4 } from 'uuid';

import ColumnPickerMenu from 'components/ColumnPickerMenu';
import ComparisonView from 'components/ComparisonView';
import { Error } from 'components/exceptions';
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
import TableFilter from 'components/FilterForm/TableFilter';
import LoadableCount from 'components/LoadableCount';
import MultiSortMenu, { EMPTY_SORT, sortMenuItemsForColumn } from 'components/MultiSortMenu';
import { OptionsMenu, RowHeight } from 'components/OptionsMenu';
import {
  DataGridGlobalSettings,
  rowHeightMap,
  settingsConfigGlobal,
} from 'components/OptionsMenu.settings';
import RunActionDropdown from 'components/RunActionDropdown';
import useUI from 'components/ThemeProvider';
import { useAsync } from 'hooks/useAsync';
import { useDebouncedSettings } from 'hooks/useDebouncedSettings';
import { useGlasbey } from 'hooks/useGlasbey';
import useMobile from 'hooks/useMobile';
import usePolling from 'hooks/usePolling';
import useResize from 'hooks/useResize';
import useScrollbarWidth from 'hooks/useScrollbarWidth';
import { useSettings } from 'hooks/useSettings';
import useTypedParams from 'hooks/useTypedParams';
import FlatRunActionButton from 'pages/FlatRuns/FlatRunActionButton';
import { paths } from 'routes/utils';
import { getProjectColumns, getProjectNumericMetricsRange, searchRuns } from 'services/api';
import { V1ColumnType, V1LocationType, V1TableType } from 'services/api-ts-sdk';
import userStore from 'stores/users';
import userSettings from 'stores/userSettings';
import {
  DetailedUser,
  FlatRun,
  FlatRunAction,
  ProjectColumn,
  RunState,
  SelectionType as SelectionState,
} from 'types';
import handleError from 'utils/error';
import { eagerSubscribe } from 'utils/observable';
import { pluralizer } from 'utils/string';

import {
  defaultColumnWidths,
  defaultRunColumns,
  defaultSearchRunColumns,
  getColumnDefs,
  RunColumn,
  runColumns,
  searcherMetricsValColumn,
} from './columns';
import css from './FlatRuns.module.scss';
import {
  ColumnWidthsSlice,
  DEFAULT_SELECTION,
  defaultFlatRunsSettings,
  FlatRunsSettings,
  ProjectUrlSettings,
  settingsPathForProject,
} from './FlatRuns.settings';

export const PAGE_SIZE = 100;
const INITIAL_LOADING_RUNS: Loadable<FlatRun>[] = new Array(PAGE_SIZE).fill(NotLoaded);

const STATIC_COLUMNS = [MULTISELECT];

const BANNED_FILTER_COLUMNS = new Set([
  'searcherMetricsVal',
  'parentArchived',
  'isExpMultitrial',
  'archived',
]);
const BANNED_SORT_COLUMNS = new Set(['tags', 'searcherMetricsVal']);

const NO_PINS_WIDTH = 200;

export const formStore = new FilterFormStore(V1LocationType.RUN);

interface Props {
  projectId: number;
  workspaceId: number;
  searchId?: number;
}

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

const FlatRuns: React.FC<Props> = ({ projectId, workspaceId, searchId }) => {
  const dataGridRef = useRef<DataGridHandle>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const { params, updateParams } = useTypedParams(ProjectUrlSettings, {});
  const page = params.page || 0;
  const setPage = useCallback(
    (p: number) => updateParams({ page: p || undefined }),
    [updateParams],
  );

  const settingsPath = useMemo(
    () => settingsPathForProject(projectId, searchId),
    [projectId, searchId],
  );
  const flatRunsSettingsObs = useMemo(
    () => userSettings.get(FlatRunsSettings, settingsPath),
    [settingsPath],
  );
  const flatRunsSettings = useObservable(flatRunsSettingsObs);
  const isLoadingSettings = useMemo(() => flatRunsSettings.isNotLoaded, [flatRunsSettings]);
  const updateSettings = useCallback(
    (p: Partial<FlatRunsSettings>) => userSettings.setPartial(FlatRunsSettings, settingsPath, p),
    [settingsPath],
  );
  const [columnWidths, updateColumnWidths] = useDebouncedSettings(ColumnWidthsSlice, settingsPath);
  const settings = useMemo(() => {
    const defaultSettings = { ...defaultFlatRunsSettings };
    if (searchId) {
      defaultSettings.columns = defaultSearchRunColumns;
    }
    return Loadable.all([flatRunsSettings, columnWidths])
      .map(([s, cw]) => ({ ...defaultSettings, ...s, ...cw }))
      .getOrElse(defaultSettings);
  }, [columnWidths, flatRunsSettings, searchId]);

  const { settings: globalSettings, updateSettings: updateGlobalSettings } =
    useSettings<DataGridGlobalSettings>(settingsConfigGlobal);

  const [isOpenFilter, setIsOpenFilter] = useState<boolean>(false);
  const [runs, setRuns] = useState<Loadable<FlatRun>[]>(INITIAL_LOADING_RUNS);

  const [sorts, setSorts] = useState<Sort[]>([EMPTY_SORT]);
  const sortString = useMemo(() => makeSortString(sorts.filter(validSort.is)), [sorts]);
  const loadableFormset = useObservable(formStore.formset);
  const rootFilterChildren: Array<FormGroup | FormField> = Loadable.match(loadableFormset, {
    _: () => [],
    Loaded: (formset: FilterFormSet) => formset.filterGroup.children,
  });
  const filtersString = useObservable(formStore.asJsonString);
  const [total, setTotal] = useState<Loadable<number>>(NotLoaded);
  const isMobile = useMobile();
  const [isLoading, setIsLoading] = useState(true);
  const [error] = useState(false);
  const [canceler] = useState(new AbortController());
  const users = useObservable<Loadable<DetailedUser[]>>(userStore.getUsers());

  const { openToast } = useToast();
  const { width: containerWidth } = useResize(contentRef);

  const {
    ui: { theme: appTheme },
    isDarkMode,
  } = useUI();

  const projectHeatmap = useAsync(async () => {
    try {
      return await getProjectNumericMetricsRange({ id: projectId });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch project heatmap' });
      return NotLoaded;
    }
  }, [projectId]);

  const projectColumns = useAsync(async () => {
    try {
      const columns = await getProjectColumns({ id: projectId, tableType: V1TableType.RUN });
      columns.sort((a, b) =>
        a.location === V1LocationType.EXPERIMENT && b.location === V1LocationType.EXPERIMENT
          ? runColumns.indexOf(a.column as RunColumn) - runColumns.indexOf(b.column as RunColumn)
          : 0,
      );
      return columns;
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch project columns' });
      return NotLoaded;
    }
  }, [projectId]);

  const arrayTypeColumns = useMemo(() => {
    const arrayTypeColumns = projectColumns
      .getOrElse([])
      .filter((col) => col.type === V1ColumnType.ARRAY)
      .map((col) => col.column);
    return arrayTypeColumns;
  }, [projectColumns]);

  const bannedFilterColumns: Set<string> = useMemo(() => {
    return new Set([...BANNED_FILTER_COLUMNS, ...arrayTypeColumns]);
  }, [arrayTypeColumns]);

  const bannedSortColumns: Set<string> = useMemo(() => {
    return new Set([...BANNED_SORT_COLUMNS, ...arrayTypeColumns]);
  }, [arrayTypeColumns]);

  const selectedRunIdSet = useMemo(() => {
    if (settings.selection.type === 'ONLY_IN') {
      return new Set(settings.selection.selections);
    } else if (settings.selection.type === 'ALL_EXCEPT') {
      const excludedSet = new Set(settings.selection.exclusions);
      return new Set(
        Loadable.filterNotLoaded(runs, (run) => !excludedSet.has(run.id)).map((run) => run.id),
      );
    }
    return new Set<number>(); // should never be reached
  }, [runs, settings.selection]);

  const columnsIfLoaded = useMemo(
    () => (isLoadingSettings ? [] : settings.columns),
    [isLoadingSettings, settings.columns],
  );

  const showPagination = useMemo(() => {
    return (
      (!settings.compare || settings.pinnedColumnsCount !== 0) && !(isMobile && settings.compare)
    );
  }, [isMobile, settings.compare, settings.pinnedColumnsCount]);

  const loadedSelectedRunIds = useMemo(() => {
    const selectedMap = new Map<number, { run: FlatRun; index: number }>();

    if (isLoadingSettings) {
      return selectedMap;
    }

    runs.forEach((r, index) => {
      Loadable.forEach(r, (run) => {
        if (selectedRunIdSet.has(run.id)) {
          selectedMap.set(run.id, { index, run });
        }
      });
    });
    return selectedMap;
  }, [isLoadingSettings, runs, selectedRunIdSet]);

  const selection = useMemo<GridSelection>(() => {
    let rows = CompactSelection.empty();
    if (settings.selection.type === 'ONLY_IN') {
      loadedSelectedRunIds.forEach((info) => {
        rows = rows.add(info.index);
      });
    } else if (settings.selection.type === 'ALL_EXCEPT') {
      rows = rows.add([0, total.getOrElse(1) - 1]);
      settings.selection.exclusions.forEach((exc) => {
        const excIndex = loadedSelectedRunIds.get(exc)?.index;
        if (excIndex !== undefined) {
          rows = rows.remove(excIndex);
        }
      });
    }
    return {
      columns: CompactSelection.empty(),
      rows,
    };
  }, [loadedSelectedRunIds, settings.selection, total]);

  const selectedRuns: FlatRun[] = useMemo(() => {
    return Loadable.filterNotLoaded(runs, (run) => selectedRunIdSet.has(run.id));
  }, [runs, selectedRunIdSet]);

  const selectionSize = useMemo(() => {
    if (settings.selection.type === 'ONLY_IN') {
      return settings.selection.selections.length;
    } else if (settings.selection.type === 'ALL_EXCEPT') {
      return total.getOrElse(0) - settings.selection.exclusions.length;
    }
    return 0;
  }, [settings.selection, total]);

  const handleIsOpenFilterChange = useCallback((newOpen: boolean) => {
    setIsOpenFilter(newOpen);
    if (!newOpen) {
      formStore.sweep();
    }
  }, []);

  const colorMap = useGlasbey([...loadedSelectedRunIds.keys()]);

  const handleToggleComparisonView = useCallback(() => {
    updateSettings({ compare: !settings.compare });
  }, [settings.compare, updateSettings]);

  const pinnedColumns = useMemo(() => {
    return [...STATIC_COLUMNS, ...settings.columns.slice(0, settings.pinnedColumnsCount)];
  }, [settings.columns, settings.pinnedColumnsCount]);

  const columns: ColumnDef<FlatRun>[] = useMemo(() => {
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
    const gridColumns = (
      settings.compare
        ? [...STATIC_COLUMNS, ...columnsIfLoaded.slice(0, settings.pinnedColumnsCount)]
        : [...STATIC_COLUMNS, ...columnsIfLoaded]
    )
      .map((columnName) => {
        if (columnName === MULTISELECT) {
          return defaultSelectionColumn(selection.rows, false);
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
          case V1LocationType.RUN:
          case V1LocationType.RUNMETADATA:
            dataPath = currentColumn.column;
            break;
          case V1LocationType.HYPERPARAMETERS:
            dataPath = `hyperparameters.${currentColumn.column.replace('hp.', '')}.val`;
            break;
          case V1LocationType.RUNHYPERPARAMETERS:
            dataPath = `hyperparameters.${currentColumn.column.replace('hp.', '')}`;
            break;
          case V1LocationType.VALIDATIONS:
            dataPath = `summaryMetrics.validationMetrics.${currentColumn.column.replace(
              'validation.',
              '',
            )}`;
            break;
          case V1LocationType.TRAINING:
            dataPath = `summaryMetrics.avgMetrics.${currentColumn.column.replace('training.', '')}`;
            break;
          case V1LocationType.CUSTOMMETRIC:
            dataPath = `summaryMetrics.${currentColumn.column}`;
            break;
          case V1LocationType.UNSPECIFIED:
          default:
            break;
        }
        switch (currentColumn.type) {
          case V1ColumnType.NUMBER: {
            const heatmap = projectHeatmap
              .getOrElse([])
              .find((h) => h.metricsName === currentColumn.column);
            if (
              heatmap &&
              settings.heatmapOn &&
              !settings.heatmapSkipped.includes(currentColumn.column)
            ) {
              columnDefs[currentColumn.column] = defaultNumberColumn(
                currentColumn.column,
                currentColumn.displayName || currentColumn.column,
                settings.columnWidths[currentColumn.column] ??
                  defaultColumnWidths[currentColumn.column as RunColumn] ??
                  MIN_COLUMN_WIDTH,
                dataPath,
                {
                  max: heatmap.max,
                  min: heatmap.min,
                },
              );
            } else {
              columnDefs[currentColumn.column] = defaultNumberColumn(
                currentColumn.column,
                currentColumn.displayName || currentColumn.column,
                settings.columnWidths[currentColumn.column] ??
                  defaultColumnWidths[currentColumn.column as RunColumn] ??
                  MIN_COLUMN_WIDTH,
                dataPath,
              );
            }
            break;
          }
          case V1ColumnType.DATE:
            columnDefs[currentColumn.column] = defaultDateColumn(
              currentColumn.column,
              currentColumn.displayName || currentColumn.column,
              settings.columnWidths[currentColumn.column] ??
                defaultColumnWidths[currentColumn.column as RunColumn] ??
                MIN_COLUMN_WIDTH,
              dataPath,
            );
            break;
          case V1ColumnType.ARRAY:
            columnDefs[currentColumn.column] = defaultArrayColumn(
              currentColumn.column,
              currentColumn.displayName || currentColumn.column,
              settings.columnWidths[currentColumn.column] ??
                defaultColumnWidths[currentColumn.column as RunColumn] ??
                MIN_COLUMN_WIDTH,
              dataPath,
            );
            break;
          case V1ColumnType.TEXT:
          case V1ColumnType.UNSPECIFIED:
          default:
            columnDefs[currentColumn.column] = defaultTextColumn(
              currentColumn.column,
              currentColumn.displayName || currentColumn.column,
              settings.columnWidths[currentColumn.column] ??
                defaultColumnWidths[currentColumn.column as RunColumn] ??
                MIN_COLUMN_WIDTH,
              dataPath,
            );
        }
        if (currentColumn.column === 'searcherMetricsVal') {
          const heatmap = projectHeatmap
            .getOrElse([])
            .find((h) => h.metricsName === currentColumn.column);

          columnDefs[currentColumn.column] = searcherMetricsValColumn(
            settings.columnWidths[currentColumn.column],
            heatmap && settings.heatmapOn && !settings.heatmapSkipped.includes(currentColumn.column)
              ? {
                  max: heatmap.max,
                  min: heatmap.min,
                }
              : undefined,
          );
        }
        return columnDefs[currentColumn.column];
      })
      .flatMap((col) => (col ? [col] : []));
    return gridColumns;
  }, [
    appTheme,
    columnsIfLoaded,
    isDarkMode,
    projectColumns,
    projectHeatmap,
    selection.rows,
    settings.columnWidths,
    settings.compare,
    settings.heatmapOn,
    settings.heatmapSkipped,
    settings.pinnedColumnsCount,
    users,
  ]);

  const onRowHeightChange = useCallback(
    (newRowHeight: RowHeight) => {
      updateGlobalSettings({ rowHeight: newRowHeight });
    },
    [updateGlobalSettings],
  );

  const handleHeatmapToggle = useCallback(
    (heatmapOn: boolean) => updateSettings({ heatmapOn: !heatmapOn }),
    [updateSettings],
  );

  const handleHeatmapSelection = useCallback(
    (selection: string[]) => updateSettings({ heatmapSkipped: selection }),
    [updateSettings],
  );

  const heatmapBtnVisible = useMemo(() => {
    const visibleColumns = settings.columns.slice(
      0,
      settings.compare ? settings.pinnedColumnsCount : undefined,
    );
    return Loadable.getOrElse([], projectColumns).some(
      (column) =>
        visibleColumns.includes(column.column) &&
        (column.column === 'searcherMetricsVal' ||
          (column.type === V1ColumnType.NUMBER &&
            (column.location === V1LocationType.VALIDATIONS ||
              column.location === V1LocationType.TRAINING))),
    );
  }, [settings.columns, projectColumns, settings.pinnedColumnsCount, settings.compare]);

  const onPageChange = useCallback(
    (cPage: number, cPageSize: number) => {
      updateSettings({ pageLimit: cPageSize });
      // Pagination component is assuming starting index of 1.
      if (cPage - 1 !== page) {
        setRuns(Array(cPageSize).fill(NotLoaded));
      }
      setPage(cPage - 1);
    },
    [page, updateSettings, setPage],
  );

  const resetPagination = useCallback(() => {
    setIsLoading(true);
    setPage(0);
    setRuns(INITIAL_LOADING_RUNS);
  }, [setPage]);

  const fetchRuns = useCallback(async (): Promise<void> => {
    if (isLoadingSettings || Loadable.isNotLoaded(loadableFormset)) return;
    try {
      const filters = JSON.parse(filtersString);
      if (searchId) {
        // only display trials for search
        const existingFilterGroup = { ...filters.filterGroup };
        const searchFilter = {
          columnName: 'experimentId',
          kind: 'field',
          location: 'LOCATION_TYPE_RUN',
          operator: '=',
          type: 'COLUMN_TYPE_NUMBER',
          value: searchId,
        };
        filters.filterGroup = {
          children: [existingFilterGroup, searchFilter],
          conjunction: 'and',
          kind: 'group',
        };
      }
      const offset = page * settings.pageLimit;
      const response = await searchRuns(
        {
          filter: JSON.stringify(filters),
          limit: settings.pageLimit,
          offset,
          projectId: projectId,
          sort: sortString || undefined,
        },
        { signal: canceler.signal },
      );
      const loadedRuns = response.runs;

      setRuns(loadedRuns.map((run) => Loaded(run)));
      setTotal(
        response.pagination.total !== undefined ? Loaded(response.pagination.total) : NotLoaded,
      );
      // if we're out of bounds, load page one
      if ((response.pagination.total || 0) < offset) {
        resetPagination();
      }
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch runs.' });
    } finally {
      setIsLoading(false);
    }
  }, [
    canceler.signal,
    filtersString,
    isLoadingSettings,
    loadableFormset,
    page,
    projectId,
    resetPagination,
    settings.pageLimit,
    sortString,
    searchId,
  ]);

  const { stopPolling } = usePolling(fetchRuns, { rerunOnNewFn: true });

  const numFilters = useMemo(() => {
    return rootFilterChildren.length;
  }, [rootFilterChildren.length]);

  useLayoutEffect(() => {
    let cleanup: () => void;
    // eslint-disable-next-line prefer-const
    cleanup = eagerSubscribe(flatRunsSettingsObs, (ps, prevPs) => {
      if (!prevPs?.isLoaded) {
        ps.forEach((s) => {
          const { sortString } = { ...defaultFlatRunsSettings, ...s };
          setSorts(parseSortString(sortString));
          cleanup?.();
        });
      }
    });
    return cleanup;
  }, [flatRunsSettingsObs]);

  useEffect(() => {
    let cleanup: () => void;
    // eagerSubscribe is like subscribe but it runs once before the observed value changes.
    cleanup = eagerSubscribe(flatRunsSettingsObs, (ps, prevPs) => {
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
              updateSettings({ filterset: JSON.stringify(formSet), selection: DEFAULT_SELECTION }),
            );
          });
        });
      }
    });
    return () => cleanup?.();
  }, [flatRunsSettingsObs, resetPagination, updateSettings]);

  const scrollbarWidth = useScrollbarWidth();

  const comparisonViewTableWidth = useMemo(() => {
    if (pinnedColumns.length === 1) return NO_PINS_WIDTH;
    return Math.min(
      containerWidth - 30,
      pinnedColumns.reduce(
        (totalWidth, curCol) =>
          totalWidth + (settings.columnWidths[curCol] ?? DEFAULT_COLUMN_WIDTH),
        scrollbarWidth,
      ),
    );
  }, [containerWidth, pinnedColumns, scrollbarWidth, settings.columnWidths]);

  const handleCompareWidthChange = useCallback(
    (newTableWidth: number) => {
      const widthDifference = newTableWidth - comparisonViewTableWidth;
      // Positive widthDifference: Table pane growing/compare pane shrinking
      // Negative widthDifference: Table pane shrinking/compare pane growing
      const newColumnWidths: Record<string, number> = {
        ...settings.columnWidths,
      };
      pinnedColumns
        .filter(
          (col) =>
            !STATIC_COLUMNS.includes(col) &&
            (widthDifference > 0 || newColumnWidths[col] > MIN_COLUMN_WIDTH),
        )
        .forEach((col, _, arr) => {
          newColumnWidths[col] = Math.max(
            MIN_COLUMN_WIDTH,
            newColumnWidths[col] + widthDifference / arr.length,
          );
        });
      updateColumnWidths({
        columnWidths: newColumnWidths,
      });
    },
    [updateColumnWidths, settings.columnWidths, pinnedColumns, comparisonViewTableWidth],
  );

  const handleColumnWidthChange = useCallback(
    (columnId: string, width: number) => {
      updateColumnWidths({
        columnWidths: { ...settings.columnWidths, [columnId]: Math.max(MIN_COLUMN_WIDTH, width) },
      });
    },
    [settings.columnWidths, updateColumnWidths],
  );

  const rowRangeToIds = useCallback(
    (range: [number, number]) => {
      const slice = runs.slice(range[0], range[1]);
      return Loadable.filterNotLoaded(slice).map((run) => run.id);
    },
    [runs],
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
            type: 'ALL_EXCEPT',
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

  const onActionComplete = useCallback(async () => {
    handleSelectionChange('remove-all');
    await fetchRuns();
  }, [fetchRuns, handleSelectionChange]);

  const handleActionSuccess = useCallback(
    (action: FlatRunAction, id: number): void => {
      const updateRun = (updated: Partial<FlatRun>) => {
        setRuns((prev) =>
          prev.map((runs) =>
            Loadable.map(runs, (run) => {
              if (run.id === id) {
                return { ...run, ...updated };
              }
              return run;
            }),
          ),
        );
      };
      switch (action) {
        case FlatRunAction.Archive:
          updateRun({ archived: true });
          break;
        case FlatRunAction.Kill:
          updateRun({ state: RunState.StoppingKilled });
          break;
        case FlatRunAction.Unarchive:
          updateRun({ archived: false });
          break;
        case FlatRunAction.Move:
        case FlatRunAction.Delete:
          setRuns((prev) =>
            prev.filter((runs) =>
              Loadable.match(runs, {
                _: () => true,
                Loaded: (run) => run.id !== id,
              }),
            ),
          );
          break;
        default:
          break;
      }
      openToast({
        severity: 'Confirm',
        title: `Run ${action.split('')[action.length - 1] === 'e' ? action.toLowerCase() : `${action.toLowerCase()}e`}d successfully`,
      });
    },
    [openToast],
  );

  const handleContextMenuComplete: ContextMenuCompleteHandlerProps<FlatRunAction, void> =
    useCallback(
      (action: FlatRunAction, id: number) => {
        handleActionSuccess(action, id);
      },
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
      updateColumnWidths({
        columnWidths: {
          ...settings.columnWidths,
          ...newColumnWidths,
        },
      });
      updateSettings({
        columns: newColumnsOrder,
        pinnedColumnsCount: pinnedCount ?? settings.pinnedColumnsCount,
      });
    },
    [updateColumnWidths, settings.columnWidths, settings.pinnedColumnsCount, updateSettings],
  );

  const handleSortChange = useCallback(
    (sorts: Sort[]) => {
      setSorts(sorts);
      const newSortString = makeSortString(sorts.filter(validSort.is));
      if (newSortString !== settings.sortString) {
        resetPagination();
      }
      updateSettings({ sortString: newSortString });
    },
    [resetPagination, settings.sortString, updateSettings],
  );

  const getRowAccentColor = (rowData: FlatRun) => {
    return colorMap[rowData.id];
  };

  const handlePinnedColumnsCountChange = useCallback(
    (newCount: number) => updateSettings({ pinnedColumnsCount: newCount }),
    [updateSettings],
  );

  const handleActualSelectAll = useCallback(() => {
    handleSelectionChange?.('add-all');
  }, [handleSelectionChange]);

  const handleClearSelect = useCallback(() => {
    handleSelectionChange?.('remove-all');
  }, [handleSelectionChange]);

  const isRangeSelected = useCallback(
    (range: [number, number]): boolean => {
      if (settings.selection.type === 'ONLY_IN') {
        const includedSet = new Set(settings.selection.selections);
        return rowRangeToIds(range).every((id) => includedSet.has(id));
      } else if (settings.selection.type === 'ALL_EXCEPT') {
        const excludedSet = new Set(settings.selection.exclusions);
        return rowRangeToIds(range).every((id) => !excludedSet.has(id));
      }
      return false; // should never be reached
    },
    [rowRangeToIds, settings.selection],
  );

  const handleHeaderClick = useCallback(
    (columnId: string): void => {
      if (columnId === MULTISELECT) {
        if (isRangeSelected([0, settings.pageLimit])) {
          handleSelectionChange?.('remove', [0, settings.pageLimit]);
        } else {
          handleSelectionChange?.('add', [0, settings.pageLimit]);
        }
      }
    },
    [handleSelectionChange, isRangeSelected, settings.pageLimit],
  );

  const getHeaderMenuItems = useCallback(
    (columnId: string, colIdx: number): MenuItem[] => {
      if (columnId === MULTISELECT) {
        return [];
      }

      const column = Loadable.getOrElse([], projectColumns).find((c) => c.column === columnId);
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
                  const newColumnsOrder = columnsIfLoaded.filter((c) => c !== columnId);
                  newColumnsOrder.splice(settings.pinnedColumnsCount, 0, columnId);
                  handleColumnsOrderChange(
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
                  const newColumnsOrder = columnsIfLoaded.filter((c) => c !== columnId);
                  newColumnsOrder.splice(settings.pinnedColumnsCount - 1, 0, columnId);
                  handleColumnsOrderChange(
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
            const newColumnsOrder = columnsIfLoaded.filter((c) => c !== columnId);
            if (isPinned) {
              handleColumnsOrderChange(
                newColumnsOrder,
                Math.max(settings.pinnedColumnsCount - 1, 0),
              );
            } else {
              handleColumnsOrderChange(newColumnsOrder);
            }
          },
        },
      ];

      if (!column) {
        return items;
      }

      if (!bannedSortColumns.has(column.column)) {
        const sortCount = sortMenuItemsForColumn(column, sorts, handleSortChange).length;
        const sortMenuItems =
          sortCount === 0
            ? []
            : [
                { type: 'divider' as const },
                ...sortMenuItemsForColumn(column, sorts, handleSortChange),
              ];
        items.push(...sortMenuItems);
      }

      const filterMenuItemsForColumn = () => {
        const isSpecialColumn = (SpecialColumnNames as ReadonlyArray<string>).includes(
          column.column,
        );
        formStore.addChild(ROOT_ID, FormKind.Field, {
          index: rootFilterChildren.length,
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

      if (!bannedFilterColumns.has(column.column)) {
        items.push(
          { type: 'divider' as const },
          {
            icon: <Icon decorative name="filter" />,
            key: 'filter',
            label: 'Add Filter',
            onClick: () => {
              setTimeout(filterMenuItemsForColumn, 5);
            },
          },
        );
      }

      const clearFilterForColumn = () => {
        formStore.removeByField(column.column);
      };

      const filterCount = formStore.getFieldCount(column.column).get();

      if (filterCount > 0) {
        items.push({
          icon: <Icon decorative name="filter" />,
          key: 'filter-clear',
          label: `Clear ${pluralizer(filterCount, 'Filter')}  (${filterCount})`,
          onClick: () => {
            setTimeout(clearFilterForColumn, 5);
          },
        });
      }

      if (
        settings.heatmapOn &&
        (column.column === 'searcherMetricsVal' ||
          (column.type === V1ColumnType.NUMBER &&
            (column.location === V1LocationType.VALIDATIONS ||
              column.location === V1LocationType.TRAINING)))
      ) {
        items.push(
          { type: 'divider' as const },
          {
            icon: <Icon decorative name="heatmap" />,
            key: 'heatmap',
            label: !settings.heatmapSkipped.includes(column.column)
              ? 'Cancel heatmap'
              : 'Apply heatmap',
            onClick: () =>
              handleHeatmapSelection?.(
                settings.heatmapSkipped.includes(column.column)
                  ? settings.heatmapSkipped.filter((p) => p !== column.column)
                  : [...settings.heatmapSkipped, column.column],
              ),
          },
        );
      }
      return items;
    },
    [
      bannedFilterColumns,
      bannedSortColumns,
      projectColumns,
      settings.pinnedColumnsCount,
      settings.heatmapOn,
      settings.heatmapSkipped,
      isMobile,
      columnsIfLoaded,
      handleColumnsOrderChange,
      rootFilterChildren,
      handleIsOpenFilterChange,
      sorts,
      handleSortChange,
      handleHeatmapSelection,
    ],
  );

  useEffect(() => {
    return () => {
      canceler.abort();
      stopPolling();
    };
  }, [canceler, stopPolling]);

  return (
    <div className={css.content} ref={contentRef}>
      <Row>
        <Column>
          <Row>
            <TableFilter
              bannedFilterColumns={bannedFilterColumns}
              entityCopy="Show runs…"
              formStore={formStore}
              isMobile={isMobile}
              isOpenFilter={isOpenFilter}
              loadableColumns={projectColumns}
              projectId={projectId}
              onIsOpenFilterChange={handleIsOpenFilterChange}
            />
            <MultiSortMenu
              bannedSortColumns={bannedSortColumns}
              columns={projectColumns}
              isMobile={isMobile}
              sorts={sorts}
              onChange={handleSortChange}
            />
            <ColumnPickerMenu
              compare={settings.compare}
              defaultPinnedCount={defaultFlatRunsSettings.pinnedColumnsCount}
              defaultVisibleColumns={searchId ? defaultSearchRunColumns : defaultRunColumns}
              initialVisibleColumns={columnsIfLoaded}
              isMobile={isMobile}
              pinnedColumnsCount={settings.pinnedColumnsCount}
              projectColumns={projectColumns}
              projectId={projectId}
              tabs={[
                V1LocationType.RUN,
                [V1LocationType.VALIDATIONS, V1LocationType.TRAINING, V1LocationType.CUSTOMMETRIC],
                V1LocationType.RUNHYPERPARAMETERS,
                V1LocationType.RUNMETADATA,
              ]}
              onVisibleColumnChange={handleColumnsOrderChange}
            />
            <OptionsMenu
              rowHeight={globalSettings.rowHeight}
              onRowHeightChange={onRowHeightChange}
            />
            <FlatRunActionButton
              isMobile={isMobile}
              projectId={projectId}
              selectedRuns={selectedRuns}
              workspaceId={workspaceId}
              onActionComplete={onActionComplete}
            />
            <LoadableCount
              labelPlural="runs"
              labelSingular="run"
              pageSize={settings.pageLimit}
              selectedCount={selectionSize}
              total={total}
              onActualSelectAll={handleActualSelectAll}
              onClearSelect={handleClearSelect}
            />
          </Row>
        </Column>
        <Column align="right">
          <Row>
            {heatmapBtnVisible && (
              <Button
                icon={<Icon name="heatmap" title="heatmap" />}
                tooltip="Toggle Metric Heatmap"
                type={settings.heatmapOn ? 'primary' : 'default'}
                onClick={() => handleHeatmapToggle(settings.heatmapOn ?? false)}
              />
            )}
            <Button
              hideChildren={isMobile}
              icon={<Icon name={settings.compare ? 'panel-on' : 'panel'} title="compare" />}
              onClick={handleToggleComparisonView}>
              Compare
            </Button>
          </Row>
        </Column>
      </Row>
      {!isLoading && total.isLoaded && total.data === 0 ? (
        numFilters === 0 ? (
          <Message
            action={
              <Link external href={paths.docs('/get-started/webui-qs.html')}>
                Quick Start Guide
              </Link>
            }
            description="Keep track of runs in a project by connecting up your code."
            icon="experiment"
            title="No Runs"
          />
        ) : (
          <Message description="No results matching your filters" icon="search" />
        )
      ) : error ? (
        <Error fetchData={fetchRuns} />
      ) : (
        <>
          <ComparisonView
            colorMap={colorMap}
            fixedColumnsCount={STATIC_COLUMNS.length + settings.pinnedColumnsCount}
            initialWidth={comparisonViewTableWidth}
            open={settings.compare}
            projectId={projectId}
            runSelection={settings.selection}
            searchId={searchId}
            tableFilters={filtersString}
            onWidthChange={handleCompareWidthChange}>
            <DataGrid<FlatRun, FlatRunAction>
              columns={columns}
              data={runs}
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
                  <RunActionDropdown
                    cell={cell}
                    link={link}
                    makeOpen={open}
                    projectId={projectId}
                    run={rowData}
                    onComplete={onComplete}
                    onLink={onClose}
                    onVisibleChange={onVisibleChange}
                  />
                );
              }}
              rowHeight={rowHeightMap[globalSettings.rowHeight as RowHeight]}
              selection={selection}
              sorts={sorts}
              staticColumns={STATIC_COLUMNS}
              onColumnResize={handleColumnWidthChange}
              onColumnsOrderChange={handleColumnsOrderChange}
              onContextMenuComplete={handleContextMenuComplete}
              onHeaderClicked={handleHeaderClick}
              onPinnedColumnsCountChange={handlePinnedColumnsCountChange}
              onSelectionChange={handleSelectionChange}
            />
          </ComparisonView>
          {showPagination && (
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
        </>
      )}
    </div>
  );
};

export default FlatRuns;
