import { CompactSelection, GridSelection, Rectangle } from '@glideapps/glide-data-grid';
import { isLeft } from 'fp-ts/lib/Either';
import Column from 'hew/Column';
import { MenuItem } from 'hew/Dropdown';
import Icon from 'hew/Icon';
import Message from 'hew/Message';
import Pagination from 'hew/Pagination';
import Row from 'hew/Row';
import { useTheme } from 'hew/Theme';
import { useToast } from 'hew/Toast';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { observable, useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { v4 as uuidv4 } from 'uuid';

import ComparisonView from 'components/ComparisonView';
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
import { RowHeight, TableViewMode } from 'components/OptionsMenu';
import TableActionBar from 'components/TableActionBar';
import useUI from 'components/ThemeProvider';
import { useGlasbey } from 'hooks/useGlasbey';
import useMobile from 'hooks/useMobile';
import usePolling from 'hooks/usePolling';
import useResize from 'hooks/useResize';
import useScrollbarWidth from 'hooks/useScrollbarWidth';
import { useSettings } from 'hooks/useSettings';
import { handlePath } from 'routes/utils';
import { getProjectColumns, getProjectNumericMetricsRange, searchExperiments } from 'services/api';
import { V1BulkExperimentFilters, V1ColumnType, V1LocationType } from 'services/api-ts-sdk';
import usersStore from 'stores/users';
import {
  ExperimentAction,
  ExperimentItem,
  ExperimentWithTrial,
  Project,
  ProjectColumn,
  ProjectMetricsRange,
  RunState,
} from 'types';
import handleError from 'utils/error';
import { getProjectExperimentForExperimentItem } from 'utils/experiment';
import { AnyMouseEvent } from 'utils/routes';
import { pluralizer } from 'utils/string';

import { Error, NoExperiments } from './exceptions';
import {
  ExperimentColumn,
  experimentColumns,
  getColumnDefs,
  searcherMetricsValColumn,
} from './expListColumns';
import css from './F_ExperimentList.module.scss';
import {
  F_ExperimentListGlobalSettings,
  F_ExperimentListSettings,
  settingsConfigForProject,
  settingsConfigGlobal,
} from './F_ExperimentList.settings';
import {
  ColumnDef,
  columnWidthsFallback,
  defaultDateColumn,
  defaultNumberColumn,
  defaultSelectionColumn,
  defaultTextColumn,
  MIN_COLUMN_WIDTH,
  MULTISELECT,
  NO_PINS_WIDTH,
} from './glide-table/columns';
import {
  ContextMenuCompleteHandlerProps,
  ContextMenuComponentProps,
} from './glide-table/contextMenu';
import GlideTable, {
  HandleSelectionChangeType,
  SCROLL_SET_COUNT_NEEDED,
  SelectionType,
  Sort,
  validSort,
  ValidSort,
} from './glide-table/GlideTable';

interface Props {
  project: Project;
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

const F_ExperimentList: React.FC<Props> = ({ project }) => {
  const contentRef = useRef<HTMLDivElement>(null);
  const [searchParams, setSearchParams] = useSearchParams();
  const settingsConfig = useMemo(() => settingsConfigForProject(project.id), [project.id]);

  const {
    isLoading: isLoadingSettings,
    settings,
    updateSettings,
  } = useSettings<F_ExperimentListSettings>(settingsConfig);
  const { settings: globalSettings, updateSettings: updateGlobalSettings } =
    useSettings<F_ExperimentListGlobalSettings>(settingsConfigGlobal);
  const isPagedView = globalSettings.tableViewMode === 'paged';
  const [page, setPage] = useState(() =>
    isFinite(Number(searchParams.get('page'))) ? Math.max(Number(searchParams.get('page')), 0) : 0,
  );
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
  const [projectColumns, setProjectColumns] = useState<Loadable<ProjectColumn[]>>(NotLoaded);
  const [projectHeatmap, setProjectHeatmap] = useState<ProjectMetricsRange[]>([]);
  const [isOpenFilter, setIsOpenFilter] = useState<boolean>(false);
  const filtersString = useObservable(formStore.asJsonString);
  const loadableFormset = useObservable(formStore.formset);
  const rootFilterChildren: Array<FormGroup | FormField> = Loadable.match(loadableFormset, {
    _: () => [],
    Loaded: (formset: FilterFormSet) => formset.filterGroup.children,
  });
  const isMobile = useMobile();
  const { openToast } = useToast();

  const [selection, setSelection] = React.useState<GridSelection>({
    columns: CompactSelection.empty(),
    rows: CompactSelection.empty(),
  });

  const selectAll = useMemo<boolean>(
    () => !isLoadingSettings && settings.selectAll,
    [isLoadingSettings, settings.selectAll],
  );
  const setSelectAll = useCallback(
    (selectAll: boolean) => {
      updateSettings({ selectAll });
    },
    [updateSettings],
  );

  const { getThemeVar } = useTheme();

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

  useEffect(() => {
    setSearchParams((params) => {
      if (page) {
        params.set('page', page.toString());
      } else {
        params.delete('page');
      }
      return params;
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, sortString]);

  useEffect(() => {
    // useSettings load the default value first, and then load the data from DB
    // use this useEffect to re-init the correct useSettings value when settings.filterset is changed
    if (isLoadingSettings) return;
    const formSetValidation = IOFilterFormSet.decode(JSON.parse(settings.filterset));
    if (isLeft(formSetValidation)) {
      handleError(formSetValidation.left, {
        publicSubject: 'Unable to initialize filterset from settings',
      });
    } else {
      const formset = formSetValidation.right;
      formStore.init(formset);
    }
  }, [settings.filterset, isLoadingSettings]);

  const [isLoading, setIsLoading] = useState(true);
  const [error] = useState(false);
  const [canceler] = useState(new AbortController());

  const colorMap = useGlasbey(settings.selectedExperiments);
  const { height: containerHeight, width: containerWidth } = useResize(contentRef);
  const height =
    containerHeight - 2 * parseInt(getThemeVar('strokeWidth')) - (isPagedView ? 40 : 0);
  const [scrollPositionSetCount] = useState(observable(0));

  const selectedExperimentIds: Set<number> = useMemo(() => {
    return isLoadingSettings ? new Set() : new Set(settings.selectedExperiments);
  }, [isLoadingSettings, settings.selectedExperiments]);

  const excludedExperimentIds: Set<number> = useMemo(() => {
    return isLoadingSettings ? new Set() : new Set(settings.excludedExperiments);
  }, [isLoadingSettings, settings.excludedExperiments]);

  useEffect(() => {
    if (isLoading) return;

    const selectedIds = new Set(selectedExperimentIds);

    if (selectAll) {
      Loadable.filterNotLoaded(experiments).forEach((experiment) => {
        const id = experiment.experiment.id;
        if (!excludedExperimentIds.has(id)) selectedIds.add(id);
      });
      updateSettings({ selectedExperiments: Array.from(selectedIds) });
    }

    /**
     * Use settings info (selectionAll, selectedExperimentIds, excludedExperimentIds)
     * to figure out and update list selections.
     */
    setSelection((prevSelection) => {
      let rows = CompactSelection.empty();
      experiments.forEach((loadable, index) => {
        const id = Loadable.getOrElse(undefined, loadable)?.experiment.id;
        if (!id) return;
        if ((selectAll && !excludedExperimentIds.has(id)) || (!selectAll && selectedIds.has(id))) {
          rows = rows.add(index);
        }
      });
      return { ...prevSelection, rows };
    });
  }, [
    excludedExperimentIds,
    experiments,
    isLoading,
    selectAll,
    selectedExperimentIds,
    total,
    updateSettings,
  ]);

  const handleScroll = useCallback(
    ({ y, height }: Rectangle) => {
      if (scrollPositionSetCount.get() < SCROLL_SET_COUNT_NEEDED) return;
      setPage(Math.floor((y + height) / PAGE_SIZE));
    },
    [scrollPositionSetCount],
  );

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

  const resetPagination = useCallback(() => {
    setIsLoading(true);
    setPage(0);
    setExperiments(INITIAL_LOADING_EXPERIMENTS);
    setSelection({ columns: CompactSelection.empty(), rows: CompactSelection.empty() });
  }, []);

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
      const tableOffset = Math.max((page - 0.5) * PAGE_SIZE, 0);
      const response = await searchExperiments(
        {
          ...experimentFilters,
          filter: filtersString,
          limit: isPagedView ? settings.pageLimit : 2 * PAGE_SIZE,
          offset: isPagedView ? page * settings.pageLimit : tableOffset,
          sort: sortString || undefined,
        },
        { signal: canceler.signal },
      );
      const total = response.pagination.total ?? 0;
      const loadedExperiments = response.experiments;

      setExperiments((prev) => {
        if (isPagedView) {
          return loadedExperiments.map((experiment) => Loaded(experiment));
        }

        let newExperiments = prev;

        // Fill out the loadable experiments array with total count.
        if (prev.length !== total) {
          newExperiments = new Array(total).fill(NotLoaded);
        }

        // Update the list with the fetched results.
        Array.prototype.splice.apply(newExperiments, [
          tableOffset,
          loadedExperiments.length,
          ...loadedExperiments.map((experiment) => Loaded(experiment)),
        ]);

        return newExperiments;
      });
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
    isPagedView,
    loadableFormset,
    page,
    sortString,
    settings.pageLimit,
  ]);

  const { stopPolling } = usePolling(fetchExperiments, { rerunOnNewFn: true });

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const heatMap = await getProjectNumericMetricsRange({ id: project.id });
        if (mounted) {
          setProjectHeatmap(heatMap);
        }
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to fetch project heatmap' });
      }
    })();
    return () => {
      mounted = false;
    };
  }, [project.id]);

  // TODO: poll?
  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const columns = await getProjectColumns({ id: project.id });
        columns.sort((a, b) =>
          a.location === V1LocationType.EXPERIMENT && b.location === V1LocationType.EXPERIMENT
            ? experimentColumns.indexOf(a.column as ExperimentColumn) -
            experimentColumns.indexOf(b.column as ExperimentColumn)
            : 0,
        );

        if (mounted) {
          setProjectColumns(Loaded(columns));
        }
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to fetch project columns' });
      }
    })();
    return () => {
      mounted = false;
    };
  }, [project.id]);

  useEffect(() => {
    return () => {
      canceler.abort();
      stopPolling();
    };
  }, [canceler, stopPolling]);

  useEffect(
    () =>
      formStore.asJsonString.subscribe(() => {
        resetPagination();
        const loadableFormset = formStore.formset.get();
        Loadable.forEach(loadableFormset, (formSet) =>
          updateSettings({ filterset: JSON.stringify(formSet) }),
        );
      }),
    [resetPagination, updateSettings],
  );

  const handleActionComplete = useCallback(async () => {
    /**
     * Deselect selected rows since their states may have changed where they
     * are no longer part of the filter criteria.
     */
    setSelection({
      columns: CompactSelection.empty(),
      rows: CompactSelection.empty(),
    });
    setSelectAll(false);

    // Re-fetch experiment list to get updates based on batch action.
    await fetchExperiments();
  }, [fetchExperiments, setSelectAll, setSelection]);

  const rowRangeToIds = useCallback(
    (range: [number, number]) => {
      return Loadable.filterNotLoaded(experiments.slice(range[0], range[1])).map(
        (experiment) => experiment.experiment.id,
      );
    },
    [experiments],
  );

  const handleSelectionChange: HandleSelectionChangeType = useCallback(
    (selectionType: SelectionType, range: [number, number]) => {
      const totalCount = Loadable.getOrElse(0, total);
      if (!totalCount) return;

      setSelection((prevSelection) => {
        const newSettings: Partial<F_ExperimentListSettings> = {};
        const excludedSet = new Set(settings.excludedExperiments);
        const includedSet = new Set(settings.selectedExperiments);
        let newSelection = prevSelection;

        switch (selectionType) {
          case 'add':
            if (selectAll) {
              rowRangeToIds(range).forEach((id) => excludedSet.delete(id));
              newSettings.excludedExperiments = Array.from(excludedSet);
            }

            rowRangeToIds(range).forEach((id) => includedSet.add(id));
            newSettings.selectedExperiments = Array.from(includedSet);

            newSelection = { ...prevSelection, rows: prevSelection.rows.add(range) };
            break;
          case 'add-all':
            newSettings.selectAll = true;
            newSettings.excludedExperiments = [];

            Loadable.filterNotLoaded(experiments).forEach((experiment) => {
              includedSet.add(experiment.experiment.id);
            });
            newSettings.selectedExperiments = Array.from(includedSet);

            newSelection = {
              columns: CompactSelection.empty(),
              rows: CompactSelection.empty().add([0, totalCount]),
            };
            break;
          case 'remove':
            if (selectAll) {
              rowRangeToIds(range).forEach((id) => excludedSet.add(id));
              newSettings.excludedExperiments = Array.from(excludedSet);
            }

            rowRangeToIds(range).forEach((id) => includedSet.delete(id));
            newSettings.selectedExperiments = Array.from(includedSet);

            newSelection = { ...prevSelection, rows: prevSelection.rows.remove(range) };
            break;
          case 'remove-all':
            newSettings.selectAll = false;
            newSettings.selectedExperiments = [];
            newSettings.excludedExperiments = [];
            newSelection = { columns: CompactSelection.empty(), rows: CompactSelection.empty() };
            break;
          case 'set':
            newSettings.selectAll = false;

            includedSet.clear();
            rowRangeToIds(range).forEach((id) => includedSet.add(id));
            newSettings.selectedExperiments = Array.from(includedSet);

            newSelection = { ...prevSelection, rows: CompactSelection.empty().add(range) };
            break;
        }

        if (Object.keys(newSettings).length !== 0) updateSettings(newSettings);

        return newSelection;
      });
    },
    [
      experiments,
      rowRangeToIds,
      selectAll,
      settings.excludedExperiments,
      settings.selectedExperiments,
      total,
      updateSettings,
    ],
  );

  const handleActionSuccess = useCallback(
    (action: ExperimentAction, successfulIds: number[], data?: Partial<ExperimentItem>): void => {
      const idSet = new Set(successfulIds);
      const updateExperiment = (updated: Partial<ExperimentItem>) => {
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
        // Exhaustive cases to ignore.
        default:
          break;
      }
      handleSelectionChange('remove-all', [0, selectedExperimentIds.size]);
    },
    [handleSelectionChange, selectedExperimentIds, openToast],
  );

  const handleContextMenuComplete: ContextMenuCompleteHandlerProps<
    ExperimentAction,
    ExperimentItem
  > = useCallback(
    (action: ExperimentAction, id: number, data?: Partial<ExperimentItem>) =>
      handleActionSuccess(action, [id], data),
    [handleActionSuccess],
  );

  const handleColumnsOrderChange = useCallback(
    (newColumnsOrder: string[]) => {
      updateSettings({ columns: newColumnsOrder });
    },
    [updateSettings],
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
        setSelectAll(false);
        setSelection({
          columns: CompactSelection.empty(),
          rows: CompactSelection.empty(),
        });
      }
    };
    window.addEventListener('keydown', handleEsc);

    return () => {
      window.removeEventListener('keydown', handleEsc);
    };
  }, [setSelectAll, setSelection]);

  const handleTableViewModeChange = useCallback(
    (mode: TableViewMode) => {
      // Reset page index when table view mode changes.
      resetPagination();
      updateGlobalSettings({ tableViewMode: mode });
    },
    [resetPagination, updateGlobalSettings],
  );

  const onPageChange = useCallback(
    (cPage: number, cPageSize: number) => {
      updateSettings({ pageLimit: cPageSize });
      // Pagination component is assuming starting index of 1.
      if (cPage - 1 !== page) {
        setExperiments(Array(cPageSize).fill(NotLoaded));
      }
      setPage(cPage - 1);
    },
    [page, updateSettings],
  );

  const handleToggleComparisonView = useCallback(() => {
    updateSettings({ compare: !settings.compare });
  }, [settings.compare, updateSettings]);

  const pinnedColumns = useMemo(() => {
    return [...STATIC_COLUMNS, ...settings.columns.slice(0, settings.pinnedColumnsCount)];
  }, [settings.columns, settings.pinnedColumnsCount]);

  const scrollbarWidth = useScrollbarWidth();

  const comparisonViewTableWidth = useMemo(() => {
    if (pinnedColumns.length === 1) return NO_PINS_WIDTH;
    return Math.min(
      containerWidth - 30,
      pinnedColumns.reduce(
        (totalWidth, curCol) =>
          totalWidth + (settings.columnWidths[curCol] ?? columnWidthsFallback),
        scrollbarWidth,
      ),
    );
  }, [containerWidth, pinnedColumns, scrollbarWidth, settings.columnWidths]);

  const handleCompareWidthChange = useCallback(
    (newTableWidth: number) => {
      const widthDifference = newTableWidth - comparisonViewTableWidth;
      // Positive widthDifference: Table pane growing/compare pane shrinking
      // Negative widthDifference: Table pane shrinking/compare pane growing
      const newColumnWidths: Record<string, number> = { ...settings.columnWidths };
      pinnedColumns
        .filter((col) => widthDifference > 0 || newColumnWidths[col] !== MIN_COLUMN_WIDTH)
        .forEach((col, _, arr) => {
          newColumnWidths[col] = Math.max(
            MIN_COLUMN_WIDTH,
            newColumnWidths[col] + widthDifference / arr.length,
          );
        });
      updateSettings({
        columnWidths: newColumnWidths,
      });
    },
    [updateSettings, settings.columnWidths, pinnedColumns, comparisonViewTableWidth],
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

  const selectedExperiments: ExperimentWithTrial[] = useMemo(() => {
    if (selectedExperimentIds.size === 0) return [];
    return Loadable.filterNotLoaded(experiments, (experiment) =>
      selectedExperimentIds.has(experiment.experiment.id),
    );
  }, [experiments, selectedExperimentIds]);

  const columnsIfLoaded = useMemo(
    () => (isLoadingSettings ? [] : settings.columns),
    [isLoadingSettings, settings.columns],
  );

  const showPagination = useMemo(() => {
    return (
      isPagedView &&
      (!settings.compare || settings.pinnedColumnsCount !== 0) &&
      !(isMobile && settings.compare)
    );
  }, [isMobile, isPagedView, settings.compare, settings.pinnedColumnsCount]);

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
    const gridColumns = (
      settings.compare
        ? [...STATIC_COLUMNS, ...columnsIfLoaded.slice(0, settings.pinnedColumnsCount)]
        : [...STATIC_COLUMNS, ...columnsIfLoaded]
    )
      .map((columnName) => {
        if (columnName === MULTISELECT) {
          return (columnDefs[columnName] = defaultSelectionColumn(selection.rows, selectAll));
        }
        if (columnName in columnDefs) return columnDefs[columnName];
        if (!Loadable.isLoaded(projectColumnsMap)) return;
        const currentColumn = projectColumnsMap.data[columnName];
        if (!currentColumn) return;
        let dataPath: string | undefined = undefined;
        switch (currentColumn.location) {
          case V1LocationType.EXPERIMENT:
            dataPath = `experiment.${currentColumn.column}`;
            break;
          case V1LocationType.HYPERPARAMETERS:
            dataPath = `experiment.config.hyperparameters.${currentColumn.column.replace(
              'hp.',
              '',
            )}.val`;
            break;
          case V1LocationType.VALIDATIONS:
            dataPath = `bestTrial.summaryMetrics.validationMetrics.${currentColumn.column.replace(
              'validation.',
              '',
            )}`;
            break;
          case V1LocationType.TRAINING:
            dataPath = `bestTrial.summaryMetrics.avgMetrics.${currentColumn.column.replace(
              'training.',
              '',
            )}`;
            break;
          case V1LocationType.CUSTOMMETRIC:
            dataPath = `bestTrial.summaryMetrics.${currentColumn.column}`;
            break;
          case V1LocationType.UNSPECIFIED:
            break;
        }
        switch (currentColumn.type) {
          case V1ColumnType.NUMBER: {
            const heatmap = projectHeatmap.find((h) => h.metricsName === currentColumn.column);
            if (
              heatmap &&
              settings.heatmapOn &&
              !settings.heatmapSkipped.includes(currentColumn.column)
            ) {
              columnDefs[currentColumn.column] = defaultNumberColumn(
                currentColumn.column,
                currentColumn.displayName || currentColumn.column,
                settings.columnWidths[currentColumn.column],
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
                settings.columnWidths[currentColumn.column],
                dataPath,
              );
            }
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
          const heatmap = projectHeatmap.find((h) => h.metricsName === currentColumn.column);
          if (
            heatmap &&
            settings.heatmapOn &&
            !settings.heatmapSkipped.includes(currentColumn.column)
          ) {
            columnDefs[currentColumn.column] = searcherMetricsValColumn(
              settings.columnWidths[currentColumn.column],
              {
                max: heatmap.max,
                min: heatmap.min,
              },
            );
          } else {
            columnDefs[currentColumn.column] = searcherMetricsValColumn(
              settings.columnWidths[currentColumn.column],
            );
          }
        }
        return columnDefs[currentColumn.column];
      })
      .flatMap((col) => (col ? [col] : []));
    return gridColumns;
  }, [
    settings.compare,
    settings.pinnedColumnsCount,
    projectColumns,
    settings.columnWidths,
    settings.heatmapSkipped,
    projectHeatmap,
    settings.heatmapOn,
    columnsIfLoaded,
    appTheme,
    isDarkMode,
    selectAll,
    selection.rows,
    users,
  ]);

  const getHeaderMenuItems = (
    columnId: string,
    colIdx: number,
    setMenuIsOpen: React.Dispatch<React.SetStateAction<boolean>>,
    scrollToTop: () => void,
    selectionRange: number,
  ): MenuItem[] => {
    if (columnId === MULTISELECT) {
      const items: MenuItem[] = [
        selection.rows.length > 0
          ? {
            key: 'select-none',
            label: 'Clear selected',
            onClick: () => {
              handleSelectionChange?.('remove-all', [0, selectionRange]);
              setMenuIsOpen(false);
            },
          }
          : null,
        ...[5, 10, 25].map((n) => ({
          key: `select-${n}`,
          label: `Select first ${n}`,
          onClick: () => {
            handleSelectionChange?.('set', [0, n]);
            scrollToTop();
            setMenuIsOpen(false);
          },
        })),
        {
          key: 'select-all',
          label: 'Select all',
          onClick: () => {
            handleSelectionChange?.('add-all', [0, selectionRange]);
            setMenuIsOpen(false);
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

    const BANNED_FILTER_COLUMNS = ['searcherMetricsVal'];
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
      setMenuIsOpen(false);
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
              handleColumnsOrderChange?.(newColumnsOrder);
              handlePinnedColumnsCountChange?.(
                Math.min(settings.pinnedColumnsCount + 1, columnsIfLoaded.length),
              );
              setMenuIsOpen(false);
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
              handleColumnsOrderChange?.(newColumnsOrder);
              handlePinnedColumnsCountChange?.(Math.max(settings.pinnedColumnsCount - 1, 0));
              setMenuIsOpen(false);
            },
          },
      {
        icon: <Icon decorative name="eye-close" />,
        key: 'hide',
        label: 'Hide column',
        onClick: () => {
          const newColumnsOrder = columnsIfLoaded.filter((c) => c !== column.column);
          handleColumnsOrderChange?.(newColumnsOrder);
          if (isPinned) {
            handlePinnedColumnsCountChange?.(Math.max(settings.pinnedColumnsCount - 1, 0));
          }
        },
      },
      { type: 'divider' as const },
      ...(BANNED_FILTER_COLUMNS.includes(column.column)
        ? []
        : [
          ...sortMenuItemsForColumn(column, sorts, handleSortChange),
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
      settings.heatmapOn &&
        (column.column === 'searcherMetricsVal' ||
          (column.type === V1ColumnType.NUMBER &&
            (column.location === V1LocationType.VALIDATIONS ||
              column.location === V1LocationType.TRAINING)))
        ? {
          icon: <Icon decorative name="heatmap" />,
          key: 'heatmap',
          label: !settings.heatmapSkipped.includes(column.column)
            ? 'Cancel heatmap'
            : 'Apply heatmap',
          onClick: () => {
            handleHeatmapSelection?.(
              settings.heatmapSkipped.includes(column.column)
                ? settings.heatmapSkipped.filter((p) => p !== column.column)
                : [...settings.heatmapSkipped, column.column],
            );
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
        compareViewOn={settings.compare}
        excludedExperimentIds={excludedExperimentIds}
        experiments={experiments}
        filters={experimentFilters}
        formStore={formStore}
        heatmapBtnVisible={heatmapBtnVisible}
        heatmapOn={settings.heatmapOn}
        initialVisibleColumns={columnsIfLoaded}
        isOpenFilter={isOpenFilter}
        project={project}
        projectColumns={projectColumns}
        rowHeight={globalSettings.rowHeight}
        selectAll={selectAll}
        selectedExperimentIds={selectedExperimentIds}
        sorts={sorts}
        tableViewMode={globalSettings.tableViewMode}
        total={total}
        onActionComplete={handleActionComplete}
        onActionSuccess={handleActionSuccess}
        onComparisonViewToggle={handleToggleComparisonView}
        onHeatmapToggle={handleHeatmapToggle}
        onIsOpenFilterChange={handleIsOpenFilterChange}
        onRowHeightChange={handleRowHeightChange}
        onSortChange={handleSortChange}
        onTableViewModeChange={handleTableViewModeChange}
        onVisibleColumnChange={handleColumnsOrderChange}
      />
      <div className={css.content} ref={contentRef}>
        {!isLoading && experiments.length === 0 ? (
          numFilters === 0 ? (
            <NoExperiments />
          ) : (
            <Message description="No results matching your filters" icon="search" />
          )
        ) : error ? (
          <Error />
        ) : (
          <div className={css.paneWrapper}>
            <ComparisonView
              fixedColumnsCount={STATIC_COLUMNS.length + settings.pinnedColumnsCount}
              initialWidth={comparisonViewTableWidth}
              open={settings.compare}
              projectId={project.id}
              selectedExperiments={selectedExperiments}
              onWidthChange={handleCompareWidthChange}>
              <GlideTable<ExperimentWithTrial, ExperimentAction, ExperimentItem>
                columns={columns}
                columnsOrder={columnsIfLoaded}
                data={experiments}
                getHeaderMenuItems={getHeaderMenuItems}
                getRowAccentColor={getRowAccentColor}
                height={height}
                hideUnpinned={settings.compare}
                numRows={isPagedView ? experiments.length : Loadable.getOrElse(PAGE_SIZE, total)}
                page={page}
                pageSize={PAGE_SIZE}
                pinnedColumnsCount={isLoadingSettings ? 0 : settings.pinnedColumnsCount}
                renderContextMenuComponent={(
                  props: ContextMenuComponentProps<
                    ExperimentWithTrial,
                    ExperimentAction,
                    ExperimentItem
                  >,
                ) => {
                  return (
                    <ExperimentActionDropdown
                      cell={props.cell}
                      experiment={getProjectExperimentForExperimentItem(
                        props.rowData.experiment,
                        project,
                      )}
                      link={props.link}
                      makeOpen={props.open}
                      onComplete={props.onComplete}
                      onLink={props.onClose}
                      onVisibleChange={props.onVisibleChange}>
                      <div />
                    </ExperimentActionDropdown>
                  );
                }}
                rowHeight={rowHeightMap[globalSettings.rowHeight as RowHeight]}
                scrollPositionSetCount={scrollPositionSetCount}
                selection={selection}
                sorts={sorts}
                staticColumns={STATIC_COLUMNS}
                onColumnResize={handleColumnWidthChange}
                onColumnsOrderChange={handleColumnsOrderChange}
                onContextMenuComplete={handleContextMenuComplete}
                onLinkClick={(href) => {
                  handlePath(event as unknown as AnyMouseEvent, { path: href });
                }}
                onPinnedColumnsCountChange={handlePinnedColumnsCountChange}
                onScroll={isPagedView ? undefined : handleScroll}
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
          </div>
        )}
      </div>
    </>
  );
};

export default F_ExperimentList;
