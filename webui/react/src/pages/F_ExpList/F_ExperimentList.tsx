import { Rectangle } from '@glideapps/glide-data-grid';
import { observable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import Page from 'components/Page';
import useResize from 'hooks/useResize';
import { searchExperiments } from 'services/api';
import { V1BulkExperimentFilters } from 'services/api-ts-sdk';
import usePolling from 'shared/hooks/usePolling';
import userStore from 'stores/users';
import { ExperimentItem, Project } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

import { defaultExperimentColumns } from './glide-table/columns';
import { Error, Loading, NoExperiments, NoMatches } from './glide-table/exceptions';
import GlideTable, { SCROLL_SET_COUNT_NEEDED } from './glide-table/GlideTable';
import TableActionBar from './glide-table/TableActionBar';
import { useGlasbey } from './useGlasbey';

interface Props {
  project: Project;
}

export const PAGE_SIZE = 100;
const F_ExperimentList: React.FC<Props> = ({ project }) => {
  const [searchParams, setSearchParams] = useSearchParams();

  const [page, setPage] = useState(
    isFinite(Number(searchParams.get('page'))) ? Number(searchParams.get('page')) : 0,
  );
  const [experiments, setExperiments] = useState<Loadable<ExperimentItem>[]>(
    Array(page * PAGE_SIZE).fill(NotLoaded),
  );
  const [total, setTotal] = useState<Loadable<number>>(NotLoaded);

  useEffect(() => {
    setSearchParams({ page: String(page) });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  const [sortableColumnIds, setSortableColumnIds] = useState(defaultExperimentColumns);
  const [selectedExperimentIds, setSelectedExperimentIds] = useState<number[]>([]);
  const [selectAll, setSelectAll] = useState(false);
  const [clearSelectionTrigger, setClearSelectionTrigger] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [error] = useState(false);
  const [canceler] = useState(new AbortController());

  const colorMap = useGlasbey(selectedExperimentIds);
  const pageRef = useRef<HTMLElement>(null);
  const { width, height } = useResize(pageRef);

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

  const fetchExperiments = useCallback(async (): Promise<void> => {
    try {
      const tableOffset = Math.max((page - 0.5) * PAGE_SIZE, 0);

      const response = await searchExperiments(
        {
          ...experimentFilters,
          limit: 2 * PAGE_SIZE,
          offset: tableOffset,
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
          ...response.experiments.map((e) => Loaded(e.experiment)),
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
  }, [page, experimentFilters, canceler.signal]);

  const { stopPolling } = usePolling(fetchExperiments, { rerunOnNewFn: true });

  useEffect(() => userStore.startPolling(), []);

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

  return (
    <Page
      bodyNoPadding
      containerRef={pageRef}
      docTitle={project.id === 1 ? 'Uncategorized Experiments' : 'Project Details'}
      id="projectDetails">
      <>
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
          <>
            <TableActionBar
              experiments={experiments}
              filters={experimentFilters}
              project={project}
              selectAll={selectAll}
              selectedExperimentIds={selectedExperimentIds}
              setExperiments={setExperiments}
              total={total}
              onAction={handleOnAction}
            />
            <GlideTable
              clearSelectionTrigger={clearSelectionTrigger}
              colorMap={colorMap}
              data={experiments}
              fetchExperiments={fetchExperiments}
              handleScroll={handleScroll}
              height={height}
              page={page}
              project={project}
              scrollPositionSetCount={scrollPositionSetCount}
              selectAll={selectAll}
              selectedExperimentIds={selectedExperimentIds}
              setSelectAll={setSelectAll}
              setSelectedExperimentIds={setSelectedExperimentIds}
              setSortableColumnIds={setSortableColumnIds}
              sortableColumnIds={sortableColumnIds}
            />
          </>
        )}
      </>
    </Page>
  );
};

export default F_ExperimentList;
