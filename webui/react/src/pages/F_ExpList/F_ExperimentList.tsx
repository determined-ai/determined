import { Rectangle } from '@glideapps/glide-data-grid';
import { observable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import Page from 'components/Page';
import useResize from 'hooks/useResize';
import { useSettings } from 'hooks/useSettings';
import { getProjectColumns, searchExperiments } from 'services/api';
import { V1BulkExperimentFilters } from 'services/api-ts-sdk';
import usePolling from 'shared/hooks/usePolling';
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

import { F_ExperimentListSettings, settingsConfigForProject } from './F_ExperimentList.settings';
import { Error, Loading, NoExperiments, NoMatches } from './glide-table/exceptions';
import GlideTable, { SCROLL_SET_COUNT_NEEDED } from './glide-table/GlideTable';
import { EMPTY_SORT, Sort, validSort, ValidSort } from './glide-table/MultiSortMenu';
import TableActionBar, { BatchAction } from './glide-table/TableActionBar';
import { useGlasbey } from './useGlasbey';

interface Props {
  project: Project;
}

const makeSortString = (sorts: ValidSort[]): string =>
  sorts.map((s) => `${s.column}=${s.direction}`).join(',');

export const PAGE_SIZE = 100;
const F_ExperimentList: React.FC<Props> = ({ project }) => {
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

  const [selectedExperimentIds, setSelectedExperimentIds] = useState<number[]>([]);
  const [selectAll, setSelectAll] = useState(false);
  const [clearSelectionTrigger, setClearSelectionTrigger] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [error] = useState(false);
  const [canceler] = useState(new AbortController());

  const colorMap = useGlasbey(selectedExperimentIds);
  const pageRef = useRef<HTMLElement>(null);
  const { width } = useResize(pageRef);
  const { height: wholePageHeight } = useResize();
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
    () => Object.values(experimentFilters).filter((x) => x !== undefined).length - 1,
    [experimentFilters],
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
  }, [page, experimentFilters, canceler.signal, sortString]);

  const { stopPolling } = usePolling(fetchExperiments, { rerunOnNewFn: true });

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

  return (
    <Page
      bodyNoPadding
      containerRef={pageRef}
      docTitle={project.id === 1 ? 'Uncategorized Experiments' : 'Project Details'}
      id="projectDetails">
      <>
        <TableActionBar
          experiments={experiments}
          filters={experimentFilters}
          handleUpdateExperimentList={handleUpdateExperimentList}
          initialVisibleColumns={settings.columns}
          project={project}
          projectColumns={projectColumns}
          selectAll={selectAll}
          selectedExperimentIds={selectedExperimentIds}
          setVisibleColumns={setVisibleColumns}
          sorts={sorts}
          total={total}
          onAction={handleOnAction}
          onSortChange={onSortChange}
        />
        {isLoading ? (
          <Loading width={width} />
        ) : experiments.length === 0 ? (
          numFilters === 0 ? (
            <NoExperiments />
          ) : (
            <NoMatches />
          )
        ) : error ? (
          <Error />
        ) : (
          <GlideTable
            clearSelectionTrigger={clearSelectionTrigger}
            colorMap={colorMap}
            data={experiments}
            fetchExperiments={fetchExperiments}
            handleScroll={handleScroll}
            handleUpdateExperimentList={handleUpdateExperimentList}
            height={wholePageHeight}
            page={page}
            project={project}
            projectColumns={projectColumns}
            scrollPositionSetCount={scrollPositionSetCount}
            selectAll={selectAll}
            selectedExperimentIds={selectedExperimentIds}
            setSelectAll={setSelectAll}
            setSelectedExperimentIds={setSelectedExperimentIds}
            setSortableColumnIds={setVisibleColumns}
            sortableColumnIds={settings.columns}
            sorts={sorts}
            onSortChange={onSortChange}
          />
        )}
      </>
    </Page>
  );
};

export default F_ExperimentList;
