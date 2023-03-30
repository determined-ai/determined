import { Rectangle } from '@glideapps/glide-data-grid';
import { Row } from 'antd';
import SkeletonButton from 'antd/es/skeleton/Button';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import Page from 'components/Page';
import useResize from 'hooks/useResize';
import { searchExperiments } from 'services/api';
import usePolling from 'shared/hooks/usePolling';
import usersStore from 'stores/users';
import { ExperimentItem, Project } from 'types';
import handleError from 'utils/error';

import GlideTable from './glide-table/GlideTable';
import { useGlasbey } from './glide-table/useGlasbey';

const experimentColumns = [
  'archived',
  'checkpointCount',
  'checkpointSize',
  'description',
  'duration',
  'forkedFrom',
  'id',
  'name',
  'progress',
  'resourcePool',
  'searcherType',
  'searcherMetricValue',
  'selected',
  'startTime',
  'state',
  'tags',
  'numTrials',
  'user',
] as const;

export type ExperimentColumn = (typeof experimentColumns)[number];

const defaultExperimentColumns: ExperimentColumn[] = [
  'id',
  'description',
  'tags',
  'forkedFrom',
  'progress',
  'startTime',
  'state',
  'searcherType',
  'user',
  'duration',
  'numTrials',
  'resourcePool',
  'checkpointSize',
  'checkpointCount',
  'searcherMetricValue',
];

interface Props {
  project: Project;
}

const emptyExperiment: Omit<ExperimentItem, 'config' | 'configRaw'> = {
  archived: false,
  checkpointCount: 0,
  checkpointSize: 0,
  description: '',
  endTime: '2021-06-21T18:11:23.756443024Z',
  forkedFrom: undefined,
  hyperparameters: {},
  id: 1,
  jobId: 'backfilled-1',
  labels: [],
  name: '',
  notes: '',
  numTrials: 0,
  projectId: 0,
  resourcePool: '',
  searcherType: 'single',
  startTime: '2020-06-26T18:09:25.844705105Z',
  state: 'COMPLETED',
  trialIds: [],
  userId: 1,
};

const PAGE_RADIUS = 50;
const F_ExperimentList: React.FC<Props> = ({ project }) => {
  const [pageMidpoint, setPageMidpoint] = useState(0);
  const [experiments, setExperiments] = useState<ExperimentItem[]>([]);
  const [sortableColumnIds, setSortableColumnIds] = useState(defaultExperimentColumns);
  const [selectedExperimentIds, setSelectedExperimentIds] = useState<string[]>([]);
  const [selectAll, setSelectAll] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [canceler] = useState(new AbortController());

  const colorMap = useGlasbey(selectedExperimentIds);
  const pageRef = useRef<HTMLElement>(null);
  const { width } = useResize(pageRef);

  const handleScroll = useCallback(({ y, height }: Rectangle) => {
    const visibleRegionMidPoint = y + height / 2;
    setPageMidpoint(Math.round(visibleRegionMidPoint / PAGE_RADIUS) * PAGE_RADIUS);
  }, []);

  const fetchExperiments = useCallback(async (): Promise<void> => {
    try {
      const tableLimit = 2 * PAGE_RADIUS;
      const tableOffset = Math.max(pageMidpoint - PAGE_RADIUS, 0);

      const response = await searchExperiments(
        {
          limit: tableLimit,
          offset: tableOffset,
          projectId: project.id,
        },
        { signal: canceler.signal },
      );

      setExperiments((prevExperiments) => {
        const paddedExperimentBeforeCurrentPage = [
          ...prevExperiments.slice(0, tableOffset),
          ...Array(Math.max(0, tableOffset - prevExperiments.length)).fill(emptyExperiment),
        ];

        const experimentsAfterCurrentPage = prevExperiments.slice(
          tableOffset + response.experiments.length,
        );
        return [
          ...paddedExperimentBeforeCurrentPage,
          ...response.experiments.map((e) => e.experiment),
          ...experimentsAfterCurrentPage,
        ];
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch experiments.' });
    } finally {
      setIsLoading(false);
    }
  }, [project.id, canceler.signal, pageMidpoint]);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([fetchExperiments(), usersStore.ensureUsersFetched(canceler)]);
  }, [fetchExperiments, canceler]);

  const { stopPolling } = usePolling(fetchAll, { rerunOnNewFn: true });

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
