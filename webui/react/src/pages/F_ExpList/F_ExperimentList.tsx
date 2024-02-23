import { CompactSelection, GridSelection, Rectangle } from '@glideapps/glide-data-grid';
import { isLeft } from 'fp-ts/lib/Either';
import Column from 'hew/Column';
import Message from 'hew/Message';
import Pagination from 'hew/Pagination';
import Row from 'hew/Row';
import { useTheme } from 'hew/Theme';
import { useToast } from 'hew/Toast';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { observable, useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import {
  FilterFormSet,
  FormField,
  FormGroup,
  IOFilterFormSet,
} from 'components/FilterForm/components/type';
import { useGlasbey } from 'hooks/useGlasbey';
import useMobile from 'hooks/useMobile';
import usePolling from 'hooks/usePolling';
import useResize from 'hooks/useResize';
import useScrollbarWidth from 'hooks/useScrollbarWidth';
import { useSettings } from 'hooks/useSettings';
import { useTypedParams } from 'hooks/useTypedParams';
import { getProjectColumns, getProjectNumericMetricsRange, searchExperiments } from 'services/api';
import { V1BulkExperimentFilters, V1ColumnType, V1LocationType } from 'services/api-ts-sdk';
import userSettings from 'stores/userSettings';
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

import ComparisonView from './ComparisonView';
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
import {
  columnWidthsFallback,
  ExperimentColumn,
  experimentColumns,
  MIN_COLUMN_WIDTH,
  MULTISELECT,
  NO_PINS_WIDTH,
} from './glide-table/columns';
import { Error, NoExperiments } from './glide-table/exceptions';
import GlideTable, { SCROLL_SET_COUNT_NEEDED, TableViewMode } from './glide-table/GlideTable';
import { EMPTY_SORT, Sort, validSort, ValidSort } from './glide-table/MultiSortMenu';
import { RowHeight } from './glide-table/OptionsMenu';
import TableActionBar from './glide-table/TableActionBar';

interface Props {
  project: Project;
}

type RangelessSelectionType = 'add-all' | 'remove-all';
type SelectionType = 'add' | 'remove' | 'set';
export interface HandleSelectionChangeType {
  (selectionType: RangelessSelectionType): void;
  (selectionType: SelectionType, range: [number, number]): void;
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

const F_ExperimentList: React.FC<Props> = ({ project }) => {
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

  const resetPagination = useCallback(() => {
    setIsLoading(true);
    setPage(0);
    setExperiments(INITIAL_LOADING_EXPERIMENTS);
  }, [setPage]);

  useEffect(() => {
    let cleanup: () => void;
    // we only want to init the filterset state from settings once and only when
    // the settings have finished loading. subscribing to the project settings
    // observable only works on initial loads -- the settings store will be
    // fully loaded when navigating to the page in-router, so we check if we
    // need to subscribe at all.
    const initFormset = (ps: Loadable<ProjectSettings | null>) => {
      ps.forEach((s) => {
        if (!s?.filterset) return;
        const formSetValidation = IOFilterFormSet.decode(JSON.parse(s.filterset));
        if (isLeft(formSetValidation)) {
          handleError(formSetValidation.left, {
            publicSubject: 'Unable to initialize filterset from settings',
          });
        } else {
          formStore.init(formSetValidation.right);
        }
        cleanup = formStore.asJsonString.subscribe(() => {
          resetPagination();
          const loadableFormset = formStore.formset.get();
          Loadable.forEach(loadableFormset, (formSet) =>
            updateSettings({ filterset: JSON.stringify(formSet) }),
          );
        });
      });
    };
    if (projectSettingsObs.get().isLoaded) {
      initFormset(projectSettingsObs.get());
    } else {
      cleanup = projectSettingsObs.subscribe((ps) => {
        if (ps.isLoaded) {
          cleanup();
          initFormset(ps);
        }
      });
    }
    return () => cleanup?.();
  }, [projectSettingsObs, resetPagination, updateSettings]);

  const [isLoading, setIsLoading] = useState(true);
  const [error] = useState(false);
  const [canceler] = useState(new AbortController());

  // partition experiment list into not selected/selected with indices and experiments so we only iterate the result list once
  const [excludedExperimentIds, selectedExperimentIds]: [
    Map<number, { index: number; experiment: ExperimentItem }>,
    Map<number, { index: number; experiment: ExperimentItem }>,
  ] = useMemo(() => {
    if (isLoadingSettings) {
      return [new Map(), new Map()];
    }
    const selectedMap = new Map();
    const excludedMap = new Map();
    const selectedIdSet =
      settings.selection.type === 'ONLY_IN' ? new Set(settings.selection.selections) : new Set();
    const excludedIdSet =
      settings.selection.type === 'ALL_EXCEPT' ? new Set(settings.selection.exclusions) : new Set();
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
  const { height: containerHeight, width: containerWidth } = useResize(contentRef);
  const height =
    containerHeight - 2 * parseInt(getThemeVar('strokeWidth')) - (isPagedView ? 40 : 0);
  const [scrollPositionSetCount] = useState(observable(0));

  const handleScroll = useCallback(
    ({ y, height }: Rectangle) => {
      if (scrollPositionSetCount.get() < SCROLL_SET_COUNT_NEEDED) return;
      setPage(Math.floor((y + height) / PAGE_SIZE));
    },
    [scrollPositionSetCount, setPage],
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
    let cleanup: () => void;
    // wait until settings is loaded before messing with it
    const handCompareOffToSettings = (ps: Loadable<ProjectSettings | null>) => {
      ps.forEach(() => {
        if (params.compare !== undefined) {
          updateSettings({ compare: params.compare });
        }
        cleanup = projectSettingsObs.subscribe((ps) => {
          ps.forEach((s) => {
            if (s) {
              updateParams({ compare: s.compare || undefined });
            }
          });
        });
      });
    };
    if (projectSettingsObs.get().isLoaded) {
      handCompareOffToSettings(projectSettingsObs.get());
    } else {
      cleanup = projectSettingsObs.subscribe((ps) => {
        if (ps.isLoaded) {
          cleanup();
          handCompareOffToSettings(ps);
        }
      });
    }
    return () => cleanup?.();
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
      handleSelectionChange('remove-all');
    },
    [handleSelectionChange, openToast],
  );

  const handleContextMenuComplete = useCallback(
    (action: ExperimentAction, id: number, data?: Partial<ExperimentItem>) =>
      handleActionSuccess(action, [id], data),
    [handleActionSuccess],
  );

  const handleVisibleColumnChange = useCallback(
    (newColumns: string[]) => {
      updateSettings({ columns: newColumns });
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
        .filter(
          (col) =>
            col !== MULTISELECT &&
            (widthDifference > 0 || newColumnWidths[col] !== MIN_COLUMN_WIDTH),
        )
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
    (newWidths: Record<string, number>) => {
      updateSettings({ columnWidths: newWidths });
    },
    [updateSettings],
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
        onVisibleColumnChange={handleVisibleColumnChange}
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
              <GlideTable
                colorMap={colorMap}
                columnWidths={settings.columnWidths}
                comparisonViewOpen={settings.compare}
                data={experiments}
                dataTotal={isPagedView ? experiments.length : Loadable.getOrElse(PAGE_SIZE, total)}
                formStore={formStore}
                heatmapOn={settings.heatmapOn}
                heatmapSkipped={settings.heatmapSkipped}
                height={height}
                page={page}
                pinnedColumnsCount={isLoadingSettings ? 0 : settings.pinnedColumnsCount}
                project={project}
                projectColumns={projectColumns}
                projectHeatmap={projectHeatmap}
                rowHeight={globalSettings.rowHeight}
                scrollPositionSetCount={scrollPositionSetCount}
                selectAll={selectAll}
                selection={selection}
                sortableColumnIds={columnsIfLoaded}
                sorts={sorts}
                staticColumns={STATIC_COLUMNS}
                onColumnResize={handleColumnWidthChange}
                onContextMenuComplete={handleContextMenuComplete}
                onHeatmapSelection={handleHeatmapSelection}
                onIsOpenFilterChange={handleIsOpenFilterChange}
                onPinnedColumnsCountChange={handlePinnedColumnsCountChange}
                onScroll={isPagedView ? undefined : handleScroll}
                onSelectionChange={handleSelectionChange}
                onSortableColumnChange={handleVisibleColumnChange}
                onSortChange={handleSortChange}
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
