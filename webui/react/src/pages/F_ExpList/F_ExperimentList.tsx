import { Rectangle } from '@glideapps/glide-data-grid';
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
import usePolling from 'hooks/usePolling';
import useResize from 'hooks/useResize';
import { useSettings } from 'hooks/useSettings';
import { getProjectColumns, searchExperiments } from 'services/api';
import { V1BulkExperimentFilters, V1LocationType } from 'services/api-ts-sdk';
import {
  ExperimentAction,
  ExperimentItem,
  ExperimentWithTrial,
  Project,
  ProjectColumn,
  RunState,
} from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { getCssVar } from 'utils/themes';

import ComparisonView from './ComparisonView';
import css from './F_ExperimentList.module.scss';
import {
  ExpListView,
  F_ExperimentListGlobalSettings,
  F_ExperimentListSettings,
  RowHeight,
  settingsConfigForProject,
  settingsConfigGlobal,
} from './F_ExperimentList.settings';
import { ExperimentColumn, experimentColumns, minColumnWidth } from './glide-table/columns';
import { Error, NoExperiments } from './glide-table/exceptions';
import GlideTable, { SCROLL_SET_COUNT_NEEDED } from './glide-table/GlideTable';
import { EMPTY_SORT, Sort, validSort, ValidSort } from './glide-table/MultiSortMenu';
import TableActionBar, { BatchAction } from './glide-table/TableActionBar';
import { useGlasbey } from './useGlasbey';

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

const STATIC_COLUMNS = ['selected', 'name'];

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
  const isPagedView = globalSettings.expListView === 'paged';
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
  const [isOpenFilter, setIsOpenFilter] = useState<boolean>(false);
  const filtersString = useObservable(formStore.asJsonString);
  const loadableFormset = useObservable(formStore.formset);
  const rootFilterChildren: Array<FormGroup | FormField> = Loadable.match(loadableFormset, {
    Loaded: (formset: FilterFormSet) => formset.filterGroup.children,
    NotLoaded: () => [],
  });

  const setPinnedColumnsCount = useCallback(
    (newCount: number) => {
      updateSettings({ pinnedColumnsCount: newCount });
    },
    [updateSettings],
  );
  const onIsOpenFilterChange = useCallback((newOpen: boolean) => {
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

  const [selectedExperimentIds, setSelectedExperimentIds] = useState<number[]>([]);
  const [excludedExperimentIds, setExcludedExperimentIds] = useState<Set<number>>(
    new Set<number>(),
  );
  const [selectAll, setSelectAll] = useState(false);
  const [clearSelectionTrigger, setClearSelectionTrigger] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [error] = useState(false);
  const [canceler] = useState(new AbortController());

  const colorMap = useGlasbey(selectedExperimentIds);
  const { height: containerHeight, width: containerWidth } = useResize(contentRef);
  const height =
    containerHeight - 2 * parseInt(getCssVar('--theme-stroke-width')) - (isPagedView ? 40 : 0);
  const [scrollPositionSetCount] = useState(observable(0));

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
  }, []);

  const onSortChange = useCallback(
    (sorts: Sort[]) => {
      setSorts(sorts);
      const newSortString = makeSortString(sorts.filter(validSort.is));
      if (newSortString !== sortString) {
        resetPagination();
      }
      updateSettings({
        sortString: newSortString,
      });
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
    page,
    experimentFilters,
    isPagedView,
    canceler.signal,
    filtersString,
    sortString,
    settings.pageLimit,
    isLoadingSettings,
    loadableFormset,
  ]);

  const { stopPolling } = usePolling(fetchExperiments, { rerunOnNewFn: true });

  const onContextMenuComplete = useCallback(fetchExperiments, [fetchExperiments]);

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

  const handleOnAction = useCallback(async () => {
    /*
     * Deselect selected rows since their states may have changed where they
     * are no longer part of the filter criteria.
     */
    setClearSelectionTrigger((prev) => prev + 1);
    setSelectAll(false);

    // Refetch experiment list to get updates based on batch action.
    await fetchExperiments();
  }, [fetchExperiments]);

  const handleUpdateExperimentList = useCallback(
    (action: BatchAction, successfulIds: number[]) => {
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
        case ExperimentAction.OpenTensorBoard:
          break;
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
      }
    },
    [setExperiments],
  );

  const setVisibleColumns = useCallback(
    (newColumns: string[]) => {
      updateSettings({ columns: newColumns });
    },
    [updateSettings],
  );

  const onRowHeightChange = useCallback(
    (newRowHeight: RowHeight) => {
      updateSettings({ rowHeight: newRowHeight });
    },
    [updateSettings],
  );

  useEffect(() => {
    const handleEsc = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setClearSelectionTrigger((prev) => prev + 1);
        setSelectAll(false);
      }
    };
    window.addEventListener('keydown', handleEsc);

    return () => {
      window.removeEventListener('keydown', handleEsc);
    };
  }, []);

  const updateExpListView = useCallback(
    (view: ExpListView) => {
      // Reset page index when table view mode changes.
      resetPagination();
      updateGlobalSettings({ expListView: view });
    },
    [resetPagination, updateGlobalSettings],
  );

  const onPageChange = useCallback(
    (cPage: number, cPageSize: number) => {
      updateSettings({ pageLimit: cPageSize });
      // Pagination component is assuming starting index of 1.
      setPage(cPage - 1);
    },
    [updateSettings],
  );

  const handleToggleComparisonView = useCallback(() => {
    updateSettings({ compare: !settings.compare });
  }, [settings.compare, updateSettings]);

  const pinnedColumns = useMemo(() => {
    return [...STATIC_COLUMNS, ...settings.columns.slice(0, settings.pinnedColumnsCount)];
  }, [settings.columns, settings.pinnedColumnsCount]);

  const comparisonViewWidth = useMemo(() => {
    return Math.min(
      containerWidth - 30,
      pinnedColumns.reduce(
        (totalWidth, curCol) => totalWidth + settings.columnWidths[curCol] ?? 0,
        17, // Constant of 17px accounts for scrollbar width
      ),
    );
  }, [containerWidth, pinnedColumns, settings.columnWidths]);

  const handleCompareWidthChange = useCallback(
    (width: number) => {
      let subtractedWidth = comparisonViewWidth - width;
      const newWidths: Record<string, number> = {};
      let iter = 1;
      while (subtractedWidth !== 0) {
        const colName = pinnedColumns[pinnedColumns.length - iter];
        const oldColWidth = settings.columnWidths[colName];
        const newColWidth = Math.max(minColumnWidth, oldColWidth - subtractedWidth);
        subtractedWidth -= oldColWidth - newColWidth;
        newWidths[colName] = newColWidth;
        iter++;
      }
      updateSettings({
        columnWidths: {
          ...settings.columnWidths,
          ...newWidths,
        },
      });
    },
    [updateSettings, settings.columnWidths, pinnedColumns, comparisonViewWidth],
  );

  const handleColumnWidthChange = useCallback(
    (newWidths: Record<string, number>) => {
      updateSettings({ columnWidths: newWidths });
    },
    [updateSettings],
  );

  const selectedExperiments: ExperimentWithTrial[] = useMemo(() => {
    if (selectedExperimentIds.length === 0) return [];
    const selectedIdSet = new Set(selectedExperimentIds);
    return Loadable.filterNotLoaded(experiments, (experiment) =>
      selectedIdSet.has(experiment.experiment.id),
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

  return (
    <>
      <TableActionBar
        excludedExperimentIds={excludedExperimentIds}
        experiments={experiments}
        expListView={globalSettings.expListView}
        filters={experimentFilters}
        formStore={formStore}
        handleUpdateExperimentList={handleUpdateExperimentList}
        initialVisibleColumns={columnsIfLoaded}
        isOpenFilter={isOpenFilter}
        project={project}
        projectColumns={projectColumns}
        rowHeight={settings.rowHeight}
        selectAll={selectAll}
        selectedExperimentIds={selectedExperimentIds}
        setExpListView={updateExpListView}
        setIsOpenFilter={onIsOpenFilterChange}
        setVisibleColumns={setVisibleColumns}
        sorts={sorts}
        toggleComparisonView={handleToggleComparisonView}
        total={total}
        onAction={handleOnAction}
        onRowHeightChange={onRowHeightChange}
        onSortChange={onSortChange}
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
              fixedColumnsCount={
                STATIC_COLUMNS.length + (isLoadingSettings ? 0 : settings.pinnedColumnsCount)
              }
              initialWidth={comparisonViewWidth}
              open={settings.compare}
              projectId={project.id}
              selectedExperiments={selectedExperiments}
              onWidthChange={handleCompareWidthChange}>
              <GlideTable
                clearSelectionTrigger={clearSelectionTrigger}
                colorMap={colorMap}
                columnWidths={settings.columnWidths}
                comparisonViewOpen={settings.compare}
                data={experimentsIfLoaded}
                dataTotal={isPagedView ? experiments.length : Loadable.getOrElse(0, total)}
                excludedExperimentIds={excludedExperimentIds}
                formStore={formStore}
                handleScroll={isPagedView ? undefined : handleScroll}
                handleUpdateExperimentList={handleUpdateExperimentList}
                height={height}
                page={page}
                pinnedColumnsCount={isLoadingSettings ? 0 : settings.pinnedColumnsCount}
                project={project}
                projectColumns={projectColumns}
                rowHeight={settings.rowHeight}
                scrollPositionSetCount={scrollPositionSetCount}
                selectAll={selectAll}
                selectedExperimentIds={selectedExperimentIds}
                setColumnWidths={handleColumnWidthChange}
                setExcludedExperimentIds={setExcludedExperimentIds}
                setPinnedColumnsCount={setPinnedColumnsCount}
                setSelectAll={setSelectAll}
                setSelectedExperimentIds={setSelectedExperimentIds}
                setSortableColumnIds={setVisibleColumns}
                sortableColumnIds={columnsIfLoaded}
                sorts={sorts}
                staticColumns={STATIC_COLUMNS}
                onContextMenuComplete={onContextMenuComplete}
                onIsOpenFilterChange={onIsOpenFilterChange}
                onSortChange={onSortChange}
              />
            </ComparisonView>
            {isPagedView && (
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
