import { Rectangle } from '@glideapps/glide-data-grid';
import { isLeft } from 'fp-ts/lib/Either';
import { observable, useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import { IOFilterFormSet } from 'components/FilterForm/components/type';
import Empty from 'components/kit/Empty';
import useResize from 'hooks/useResize';
import { useSettings } from 'hooks/useSettings';
import { getProjectColumns, searchExperiments } from 'services/api';
import { V1BulkExperimentFilters } from 'services/api-ts-sdk';
import usePolling from 'shared/hooks/usePolling';
import { getCssVar } from 'shared/themes';
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

import ComparisonView from './ComparisonView';
import css from './F_ExperimentList.module.scss';
import { F_ExperimentListSettings, settingsConfigForProject } from './F_ExperimentList.settings';
import { Error, Loading, NoExperiments } from './glide-table/exceptions';
import GlideTable, { SCROLL_SET_COUNT_NEEDED } from './glide-table/GlideTable';
import { EMPTY_SORT, Sort, validSort, ValidSort } from './glide-table/MultiSortMenu';
import TableActionBar, { BatchAction } from './glide-table/TableActionBar';
import { useGlasbey } from './useGlasbey';

interface Props {
  project: Project;
}

const makeSortString = (sorts: ValidSort[]): string =>
  sorts.map((s) => `${s.column}=${s.direction}`).join(',');

const formStore = new FilterFormStore();

export const PAGE_SIZE = 100;

const F_ExperimentList: React.FC<Props> = ({ project }) => {
  const contentRef = useRef<HTMLDivElement>(null);
  const [searchParams, setSearchParams] = useSearchParams();
  const settingsConfig = useMemo(() => settingsConfigForProject(project.id), [project.id]);

  const { settings, updateSettings } = useSettings<F_ExperimentListSettings>(settingsConfig);

  const [page, setPage] = useState(() =>
    isFinite(Number(searchParams.get('page'))) ? Number(searchParams.get('page')) : 0,
  );
  const [sorts, setSorts] = useState<Sort[]>(() => {
    const sortString = searchParams.get('sort') || '';
    if (!sortString) {
      return [EMPTY_SORT];
    }
    const components = sortString.split(',');
    return components.map((c) => {
      const [column, direction] = c.split('=', 2);
      return {
        column,
        direction: direction === 'asc' || direction === 'desc' ? direction : undefined,
      };
    });
  });
  const [sortString, setSortString] = useState<string>('');
  const [experiments, setExperiments] = useState<Loadable<ExperimentWithTrial>[]>(
    Array(page * PAGE_SIZE).fill(NotLoaded),
  );
  const [total, setTotal] = useState<Loadable<number>>(NotLoaded);
  const [projectColumns, setProjectColumns] = useState<Loadable<ProjectColumn[]>>(NotLoaded);
  const [isOpenFilter, setIsOpenFilter] = useState<boolean>(false);
  const filtersString = useObservable(formStore.asJsonString);
  const rootFilterChildren = useObservable(formStore.formset).filterGroup.children;

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
      if (sortString) {
        params.set('sort', sortString);
      } else {
        params.delete('sort');
      }
      return params;
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, sortString]);

  useEffect(() => {
    // useSettings load the default value first, and then load the data from DB
    // use this useEffect to re-init the correct useSettings value when settings.filterset is changed
    const formSetValidation = IOFilterFormSet.decode(JSON.parse(settings.filterset));
    if (isLeft(formSetValidation)) {
      handleError(formSetValidation.left, {
        publicSubject: 'Unable to initialize filterset from settings',
      });
    } else {
      const formset = formSetValidation.right;
      formStore.init(formset);
    }
  }, [settings.filterset]);

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
  const { height, width } = useResize(contentRef);
  const [scrollPositionSetCount] = useState(observable(0));

  const handleScroll = useCallback(
    ({ y, height }: Rectangle) => {
      if (scrollPositionSetCount.get() < SCROLL_SET_COUNT_NEEDED) return;
      const page = Math.floor((y + height) / PAGE_SIZE);
      setPage(page);
    },
    [scrollPositionSetCount],
  );

  const experimentFilters = useMemo(() => {
    const filters: V1BulkExperimentFilters = {
      projectId: project.id,
    };
    return filters;
  }, [project.id]);

  const numFilters = useMemo(
    () =>
      Object.values(experimentFilters).filter((x) => x !== undefined).length -
      1 +
      rootFilterChildren.length,
    [experimentFilters, rootFilterChildren.length],
  );

  const resetPagination = useCallback(() => {
    setIsLoading(true);
    setPage(0);
    setExperiments([]);
  }, []);

  const onSortChange = useCallback(
    (sorts: Sort[]) => {
      setSorts(sorts);
      const newSortString = makeSortString(sorts.filter(validSort.is));
      if (newSortString !== sortString) {
        resetPagination();
      }
      setSortString(newSortString);
    },
    [resetPagination, sortString],
  );

  const fetchExperiments = useCallback(async (): Promise<void> => {
    try {
      const tableOffset = Math.max((page - 0.5) * PAGE_SIZE, 0);

      const response = await searchExperiments(
        {
          ...experimentFilters,
          filter: filtersString,
          limit: 2 * PAGE_SIZE,
          offset: tableOffset,
          sort: sortString || undefined,
        },
        { signal: canceler.signal },
      );

      setExperiments((prevExperiments) => {
        const experimentBeforeCurrentPage = [
          ...prevExperiments.slice(0, tableOffset),
          ...Array(Math.max(0, tableOffset - prevExperiments.length)).fill(NotLoaded),
        ];

        const experimentsAfterCurrentPage = prevExperiments.slice(
          tableOffset + response.experiments.length,
        );
        return [
          ...experimentBeforeCurrentPage,
          ...response.experiments.map((e) => Loaded(e)),
          ...experimentsAfterCurrentPage,
        ].slice(0, response.pagination.total);
      });
      setTotal(
        response.pagination.total !== undefined ? Loaded(response.pagination.total) : NotLoaded,
      );
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch experiments.' });
    } finally {
      setIsLoading(false);
    }
  }, [page, experimentFilters, canceler.signal, filtersString, sortString]);

  const { stopPolling } = usePolling(fetchExperiments, { rerunOnNewFn: true });

  const onContextMenuComplete = useCallback(fetchExperiments, [fetchExperiments]);

  // TODO: poll?
  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const columns = await getProjectColumns({ id: project.id });

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
      updateSettings({ filterset: JSON.stringify(formStore.formset.get()) });
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

  const handleToggleComparisonView = useCallback(() => {
    updateSettings({ compare: !settings.compare });
  }, [settings.compare, updateSettings]);

  const handleCompareWidthChange = useCallback(
    (width: number) => {
      updateSettings({ compareWidth: width });
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

  return (
    <>
      <TableActionBar
        excludedExperimentIds={excludedExperimentIds}
        experiments={experiments}
        filters={experimentFilters}
        formStore={formStore}
        handleUpdateExperimentList={handleUpdateExperimentList}
        initialVisibleColumns={settings.columns}
        isOpenFilter={isOpenFilter}
        project={project}
        projectColumns={projectColumns}
        selectAll={selectAll}
        selectedExperimentIds={selectedExperimentIds}
        setIsOpenFilter={onIsOpenFilterChange}
        setVisibleColumns={setVisibleColumns}
        sorts={sorts}
        toggleComparisonView={handleToggleComparisonView}
        total={total}
        onAction={handleOnAction}
        onSortChange={onSortChange}
      />
      <div className={css.content} ref={contentRef}>
        {isLoading ? (
          <Loading width={width} />
        ) : experiments.length === 0 ? (
          numFilters === 0 ? (
            <NoExperiments />
          ) : (
            <Empty description="No results matching your filters" icon="search" />
          )
        ) : error ? (
          <Error />
        ) : (
          <ComparisonView
            initialWidth={settings.compareWidth}
            open={settings.compare}
            selectedExperiments={selectedExperiments}
            onWidthChange={handleCompareWidthChange}>
            <GlideTable
              clearSelectionTrigger={clearSelectionTrigger}
              colorMap={colorMap}
              data={experiments}
              dataTotal={Loadable.getOrElse(0, total)}
              excludedExperimentIds={excludedExperimentIds}
              formStore={formStore}
              handleScroll={handleScroll}
              handleUpdateExperimentList={handleUpdateExperimentList}
              height={height - 2 * parseInt(getCssVar('--theme-stroke-width'))}
              page={page}
              project={project}
              projectColumns={projectColumns}
              scrollPositionSetCount={scrollPositionSetCount}
              selectAll={selectAll}
              selectedExperimentIds={selectedExperimentIds}
              setExcludedExperimentIds={setExcludedExperimentIds}
              setSelectAll={setSelectAll}
              setSelectedExperimentIds={setSelectedExperimentIds}
              setSortableColumnIds={setVisibleColumns}
              sortableColumnIds={settings.columns}
              sorts={sorts}
              onContextMenuComplete={onContextMenuComplete}
              onIsOpenFilterChange={onIsOpenFilterChange}
              onSortChange={onSortChange}
            />
          </ComparisonView>
        )}
      </div>
    </>
  );
};

export default F_ExperimentList;
