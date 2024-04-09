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
  MIN_COLUMN_WIDTH,
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
import Message from 'hew/Message';
import Pagination from 'hew/Pagination';
import Row from 'hew/Row';
import { useToast } from 'hew/Toast';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { v4 as uuidv4 } from 'uuid';

import ComparisonView from 'components/ComparisonView';
import { Error, NoExperiments } from 'components/exceptions';
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
import { useTypedParams } from 'hooks/useTypedParams';
import { getProjectColumns, getProjectNumericMetricsRange, searchExperiments } from 'services/api';
import { V1BulkExperimentFilters, V1ColumnType, V1LocationType } from 'services/api-ts-sdk';
import usersStore from 'stores/users';
import userSettings from 'stores/userSettings';
import {
  BulkExperimentItem,
  ExperimentAction,
  ExperimentWithTrial,
  Project,
  ProjectColumn,
  ProjectMetricsRange,
  RunState,
} from 'types';
import handleError from 'utils/error';
import { getProjectExperimentForExperimentItem } from 'utils/experiment';
import { eagerSubscribe } from 'utils/observable';
import { pluralizer } from 'utils/string';

import {
  ExperimentColumn,
  experimentColumns,
  getColumnDefs,
  searcherMetricsValColumn,
} from './expListColumns';
import css from './F_ExperimentList.module.scss';
import {
  DEFAULT_SELECTION,
  defaultProjectSettings,
  F_ExperimentListGlobalSettings,
  ProjectSettings,
  ProjectUrlSettings,
  SelectionType as SelectionState,
  settingsConfigGlobal,
  settingsPathForProject,
} from './F_ExperimentList.settings';

interface Props {
  project: Project;
}

type ExperimentWithIndex = { index: number; experiment: BulkExperimentItem };

const NO_PINS_WIDTH = 200;

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
    useSettings<F_ExperimentListGlobalSettings>(settingsConfigGlobal);
  const isPagedView = globalSettings.tableViewMode === 'paged';
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

  const selectAll = useMemo<boolean>(
    () => !isLoadingSettings && settings.selection.type === 'ALL_EXCEPT',
    [isLoadingSettings, settings.selection],
  );

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
              updateSettings({ filterset: JSON.stringify(formSet) }),
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

  // partition experiment list into not selected/selected with indices and experiments so we only iterate the result list once
  const [excludedExperimentIds, selectedExperimentIds] = useMemo(() => {
    const selectedMap = new Map<number, ExperimentWithIndex>();
    const excludedMap = new Map<number, ExperimentWithIndex>();
    if (isLoadingSettings) {
      return [excludedMap, selectedMap];
    }
    const selectedIdSet = new Set(
      settings.selection.type === 'ONLY_IN' ? settings.selection.selections : [],
    );
    const excludedIdSet = new Set(
      settings.selection.type === 'ALL_EXCEPT' ? settings.selection.exclusions : [],
    );
    experiments.forEach((e, index) => {
      Loadable.forEach(e, ({ experiment }) => {
        const mapToAdd =
          (selectAll && !excludedIdSet.has(experiment.id)) || selectedIdSet.has(experiment.id)
            ? selectedMap
            : excludedMap;
        mapToAdd.set(experiment.id, { experiment, index });
      });
    });
    return [excludedMap, selectedMap];
  }, [isLoadingSettings, selectAll, settings.selection, experiments]);

  const selection = useMemo<GridSelection>(() => {
    let rows = CompactSelection.empty();
    if (selectAll) {
      Loadable.forEach(total, (t) => {
        rows = rows.add([0, t]);
      });
      excludedExperimentIds.forEach((info) => {
        rows = rows.remove(info.index);
      });
    } else {
      selectedExperimentIds.forEach((info) => {
        rows = rows.add(info.index);
      });
    }
    return {
      columns: CompactSelection.empty(),
      rows,
    };
  }, [selectAll, selectedExperimentIds, excludedExperimentIds, total]);

  const colorMap = useGlasbey([...selectedExperimentIds.keys()]);
  const { width: containerWidth } = useResize(contentRef);

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

  useEffect(() => {
    return eagerSubscribe(projectSettingsObs, (ps, prevPs) => {
      if (!prevPs?.isLoaded) {
        ps.forEach(() => {
          if (params.compare !== undefined) {
            updateSettings({ compare: params.compare });
          }
        });
      } else {
        ps.forEach((s) => {
          if (s) {
            updateParams({ compare: s.compare || undefined });
          }
        });
      }
    });
  }, [params.compare, updateSettings, updateParams, projectSettingsObs]);

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
        handleSelectionChange('remove-all');
      }
    };
    window.addEventListener('keydown', handleEsc);

    return () => {
      window.removeEventListener('keydown', handleEsc);
    };
  }, [handleSelectionChange]);

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
    [page, updateSettings, setPage],
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
            dataPath = `experiment.hyperparameters.${currentColumn.column.replace('hp.', '')}.val`;
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
          default:
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

  const getHeaderMenuItems = (columnId: string, colIdx: number): MenuItem[] => {
    if (columnId === MULTISELECT) {
      const items: MenuItem[] = [
        selection.rows.length > 0
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
            handleSelectionChange?.('add-all');
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
        columnGroups={[
          V1LocationType.EXPERIMENT,
          [V1LocationType.VALIDATIONS, V1LocationType.TRAINING, V1LocationType.CUSTOMMETRIC],
          V1LocationType.HYPERPARAMETERS,
        ]}
        compareViewOn={settings.compare}
        excludedExperimentIds={excludedExperimentIds}
        experiments={experiments}
        filters={experimentFilters}
        formStore={formStore}
        heatmapBtnVisible={heatmapBtnVisible}
        heatmapOn={settings.heatmapOn}
        initialVisibleColumns={columnsIfLoaded}
        isOpenFilter={isOpenFilter}
        labelPlural="experiments"
        labelSingular="experiment"
        project={project}
        projectColumns={projectColumns}
        rowHeight={globalSettings.rowHeight}
        selectAll={selectAll}
        selectedExperimentIds={selectedExperimentIds}
        selection={settings.selection}
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
              <DataGrid<ExperimentWithTrial, ExperimentAction, BulkExperimentItem>
                columns={columns}
                data={experiments}
                getHeaderMenuItems={getHeaderMenuItems}
                getRowAccentColor={getRowAccentColor}
                hideUnpinned={settings.compare}
                imperativeRef={dataGridRef}
                isPaginated={isPagedView}
                page={page}
                pageSize={PAGE_SIZE}
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
                      experiment={getProjectExperimentForExperimentItem(
                        rowData.experiment,
                        project,
                      )}
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
                total={Loadable.getOrElse(PAGE_SIZE, total)}
                onColumnResize={handleColumnWidthChange}
                onColumnsOrderChange={handleColumnsOrderChange}
                onContextMenuComplete={handleContextMenuComplete}
                onPageUpdate={setPage}
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
          </div>
        )}
      </div>
    </>
  );
};

export default F_ExperimentList;
