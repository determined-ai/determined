import { Rectangle } from '@glideapps/glide-data-grid';
import { Row } from 'antd';
import SkeletonButton from 'antd/es/skeleton/Button';
import { observable } from 'micro-observables';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import Page from 'components/Page';
import useResize from 'hooks/useResize';
import { searchExperiments } from 'services/api';
import usePolling from 'shared/hooks/usePolling';
import userStore from 'stores/users';
import { ExperimentItem, Project } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

import { defaultExperimentColumns } from './glide-table/columns';
import GlideTable from './glide-table/GlideTable';
import { useGlasbey } from './glide-table/useGlasbey';

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

  useEffect(() => {
    setSearchParams({ page: String(page) });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  const [sortableColumnIds, setSortableColumnIds] = useState(defaultExperimentColumns);
  const [selectedExperimentIds, setSelectedExperimentIds] = useState<string[]>([]);
  const [selectAll, setSelectAll] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [canceler] = useState(new AbortController());

  const colorMap = useGlasbey(selectedExperimentIds);
  const pageRef = useRef<HTMLElement>(null);
  const { width } = useResize(pageRef);

  const [initialScrollPositionSet] = useState(observable(false));

  const handleScroll = useCallback(
    ({ y, height }: Rectangle) => {
      if (!initialScrollPositionSet.get()) return;
      const page = Math.floor((y + height) / PAGE_SIZE);
      setPage(page);
    },
    [initialScrollPositionSet],
  );

  const fetchExperiments = useCallback(async (): Promise<void> => {
    try {
      const tableOffset = Math.max((page - 0.5) * PAGE_SIZE, 0);

      const response = await searchExperiments(
        {
          limit: 2 * PAGE_SIZE,
          offset: tableOffset,
          orderBy: 'ORDER_BY_DESC',
          projectId: project.id,
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
        ];
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch experiments.' });
    } finally {
      setIsLoading(false);
    }
  }, [project.id, canceler.signal, page]);

  const { stopPolling } = usePolling(fetchExperiments, { rerunOnNewFn: true });

  useEffect(() => userStore.startPolling(), []);

  useEffect(() => {
    return () => {
      canceler.abort();
      stopPolling();
    };
  }, [canceler, stopPolling]);

  return (
    <Page
      bodyNoPadding
      containerRef={pageRef}
      docTitle={project.id === 1 ? 'Uncategorized Experiments' : 'Project Details'}
      id="projectDetails">
      <>
        {isLoading ? (
          [...Array(22)].map((x, i) => (
            <Row key={i} style={{ paddingBottom: '4px' }}>
              <SkeletonButton style={{ width: width - 20 }} />
            </Row>
          ))
        ) : (
          <GlideTable
            colorMap={colorMap}
            data={experiments}
            handleScroll={handleScroll}
            initialScrollPositionSet={initialScrollPositionSet}
            page={page}
            selectAll={selectAll}
            selectedExperimentIds={selectedExperimentIds}
            setSelectAll={setSelectAll}
            setSelectedExperimentIds={setSelectedExperimentIds}
            setSortableColumnIds={setSortableColumnIds}
            sortableColumnIds={sortableColumnIds}
          />
        )}
      </>
    </Page>
  );
};

export default F_ExperimentList;
