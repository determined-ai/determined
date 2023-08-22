import { CompactSelection, GridSelection, Rectangle } from '@hpe.com/glide-data-grid';
import { Space } from 'antd';
import { isLeft } from 'fp-ts/lib/Either';
import { observable, useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import {
  FilterFormSet,
  FormField,
  FormGroup,
  IOFilterFormSet,
} from 'components/FilterForm/components/type';
import { Column, Columns } from 'components/kit/Columns';
import Empty from 'components/kit/Empty';
import Pagination from 'components/kit/Pagination';
import { useGlasbey } from 'hooks/useGlasbey';
import useMobile from 'hooks/useMobile';
import usePolling from 'hooks/usePolling';
import useResize from 'hooks/useResize';
import useScrollbarWidth from 'hooks/useScrollbarWidth';
import { useSettings } from 'hooks/useSettings';
import { getProjectColumns, getProjectNumericMetricsRange, searchExperiments } from 'services/api';
import { V1BulkExperimentFilters, V1ColumnType, V1LocationType } from 'services/api-ts-sdk';
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
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { getCssVar } from 'utils/themes';

import ComparisonView from './ComparisonView';
import css from './F_ExperimentList.module.scss';
import {
  F_ExperimentListGlobalSettings,
  F_ExperimentListSettings,
  settingsConfigForProject,
  settingsConfigGlobal,
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
    Loaded: (formset: FilterFormSet) => formset.filterGroup.children,
    NotLoaded: () => [],
  });
  const isMobile = useMobile();

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
    containerHeight - 2 * parseInt(getCssVar('--theme-stroke-width')) - (isPagedView ? 40 : 0);
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
    if (isLoadingSettings || Loadable.isLoading(loadableFormset)) return;
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

  useEffect(() => {
    return formStore.asJsonString.subscribe(() => {
      resetPagination();
      const loadableFormset = formStore.formset.get();
      Loadable.forEach(loadableFormset, (formSet) =>
        updateSettings({ filterset: JSON.stringify(formSet) }),
      );
    });
  }, [resetPagination, updateSettings]);

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

  const handleActionSuccess = useCallback(
    (action: ExperimentAction, successfulIds: number[]) => {
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
        case ExperimentAction.Move:
        case ExperimentAction.Delete:
          setExperiments((prev) =>
            prev.filter((expLoadable) =>
              Loadable.match(expLoadable, {
                Loaded: (experiment) => !idSet.has(experiment.experiment.id),
                NotLoaded: () => true,
              }),
            ),
          );
          break;
        // Exhaustive cases to ignore.
        case ExperimentAction.CompareTrials:
        case ExperimentAction.ContinueTrial:
        case ExperimentAction.DownloadCode:
        case ExperimentAction.Edit:
        case ExperimentAction.Fork:
        case ExperimentAction.HyperparameterSearch:
        case ExperimentAction.OpenTensorBoard:
        case ExperimentAction.SwitchPin:
        case ExperimentAction.ViewLogs:
          break;
      }
    },
    [setExperiments],
  );

  const handleContextMenuComplete = useCallback(
    (action: ExperimentAction, id: number) => handleActionSuccess(action, [id]),
    [handleActionSuccess],
  );

  const rowRangeToIds = useCallback(
    (range: [number, number]) => {
      return Loadable.filterNotLoaded(experiments.slice(range[0], range[1])).map(
        (experiment) => experiment.experiment.id,
      );
    },
    [experiments],
  );

  const handleSelectionChange = useCallback(
    (
      selectionType: 'add' | 'add-all' | 'remove' | 'remove-all' | 'set',
      range: [number, number],
    ) => {
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

  const experimentsIfLoaded = useMemo(
    () => (isLoading ? [NotLoaded] : experiments),
    [isLoading, experiments],
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
            <Empty description="No results matching your filters" icon="search" />
          )
        ) : error ? (
          <Error />
        ) : (
          <Space className={css.space} direction="vertical">
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
                data={experimentsIfLoaded}
                dataTotal={isPagedView ? experiments.length : Loadable.getOrElse(0, total)}
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
              <Columns>
                <Column align="right">
                  <Pagination
                    current={page + 1}
                    pageSize={settings.pageLimit}
                    pageSizeOptions={[20, 40, 80]}
                    total={Loadable.getOrElse(0, total)}
                    onChange={onPageChange}
                  />
                </Column>
              </Columns>
            )}
          </Space>
        )}
      </div>
    </>
  );
};

export default F_ExperimentList;
