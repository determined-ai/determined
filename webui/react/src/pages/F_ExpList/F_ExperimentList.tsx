import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Rectangle } from '@glideapps/glide-data-grid';
import { Row } from 'antd';
import SkeletonButton from 'antd/es/skeleton/Button';
import useModalExperimentMove from 'hooks/useModal/Experiment/useModalExperimentMove';
import { observable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import Page from 'components/Page';
import usePermissions from 'hooks/usePermissions';
import useResize from 'hooks/useResize';
import {
  activateExperiments,
  archiveExperiments,
  cancelExperiments,
  getExperiments,
  killExperiments,
  openOrCreateTensorBoard,
  pauseExperiments,
  unarchiveExperiments,
} from 'services/api';
import { V1BulkExperimentFilters, V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import usePolling from 'shared/hooks/usePolling';
import { RecordKey } from 'shared/types';
import { ErrorLevel } from 'shared/utils/error';
import userStore from 'stores/users';
import {
  ExperimentAction as Action,
  BulkActionError,
  CommandResponse,
  ExperimentItem,
  Project,
  ProjectExperiment,
} from 'types';
import { modal } from 'utils/dialogApi';
import handleError from 'utils/error';
import {
  canActionExperiment,
  getActionsForExperimentsUnion,
  getProjectExperimentForExperimentItem,
} from 'utils/experiment';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

import { defaultExperimentColumns } from './glide-table/columns';
import GlideTable from './glide-table/GlideTable';
import TableActionBar from './glide-table/TableActionBar';
import { useGlasbey } from './glide-table/useGlasbey';

interface Props {
  project: Project;
}

const batchActions = [
  Action.OpenTensorBoard,
  Action.Activate,
  Action.Move,
  Action.Pause,
  Action.Archive,
  Action.Unarchive,
  Action.Cancel,
  Action.Kill,
  Action.Delete,
];

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
  const [selectedExperimentIds, setSelectedExperimentIds] = useState<number[]>([]);
  const [selectAll, setSelectAll] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [canceler] = useState(new AbortController());

  const permissions = usePermissions();
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

  const fetchFilters: V1BulkExperimentFilters = useMemo(
    () => ({
      archived: false,
      limit: 2 * PAGE_SIZE,
      orderBy: 'ORDER_BY_DESC',
      projectId: project.id,
      sortBy: V1GetExperimentsRequestSortBy.ID,
    }),
    [project.id],
  );

  const fetchExperiments = useCallback(async (): Promise<void> => {
    try {
      const tableOffset = Math.max((page - 0.5) * PAGE_SIZE, 0);

      const response = await getExperiments(
        {
          ...fetchFilters,
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
          ...response.experiments.map((e) => Loaded(e)),
          ...experimentsAfterCurrentPage,
        ];
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch experiments.' });
    } finally {
      setIsLoading(false);
    }
  }, [page, fetchFilters, canceler.signal]);

  const { stopPolling } = usePolling(fetchExperiments, { rerunOnNewFn: true });

  useEffect(() => userStore.startPolling(), []);

  useEffect(() => {
    return () => {
      canceler.abort();
      stopPolling();
    };
  }, [canceler, stopPolling]);

  const { contextHolder: modalExperimentMoveContextHolder, modalOpen: openMoveModal } =
    useModalExperimentMove({ onClose: fetchAll });

  const experimentMap = useMemo(() => {
    return experiments.filter(Loadable.isLoaded).reduce((acc, experiment) => {
      acc[experiment.data.id] = getProjectExperimentForExperimentItem(experiment.data, project);
      return acc;
    }, {} as Record<RecordKey, ProjectExperiment>);
  }, [experiments, project]);

  const availableBatchActions = useMemo(() => {
    const experiments = selectedExperimentIds.map((id) => experimentMap[id]) ?? [];
    return getActionsForExperimentsUnion(experiments, batchActions, permissions);
  }, [experimentMap, permissions, selectedExperimentIds]);

  const sendBatchActions = useCallback(
    (action: Action): Promise<BulkActionError[] | void | CommandResponse> | void => {
      const selectedIds = selectedExperimentIds;
      if (action === Action.OpenTensorBoard) {
        return openOrCreateTensorBoard({
          experimentIds: selectedIds,
          filters: selectAll ? fetchFilters : undefined,
          workspaceId: project?.workspaceId,
        });
      }
      if (action === Action.Move) {
        return openMoveModal({
          experimentIds: selectedExperimentIds.filter(
            (id) =>
              canActionExperiment(Action.Move, experimentMap[id]) &&
              permissions.canMoveExperiment({ experiment: experimentMap[id] }),
          ),
          filters: selectAll ? fetchFilters : undefined,
          sourceProjectId: project?.id,
          sourceWorkspaceId: project?.workspaceId,
        });
      }

      switch (action) {
        case Action.Activate:
          return activateExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? fetchFilters : undefined,
          });
        case Action.Archive:
          return archiveExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? fetchFilters : undefined,
          });
        case Action.Cancel:
          return cancelExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? fetchFilters : undefined,
          });
        case Action.Kill:
          return killExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? fetchFilters : undefined,
          });
        case Action.Pause:
          return pauseExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? fetchFilters : undefined,
          });
        case Action.Unarchive:
          return unarchiveExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? fetchFilters : undefined,
          });
        default:
          return Promise.resolve();
      }
    },
    [
      openMoveModal,
      selectedExperimentIds,
      project?.id,
      project?.workspaceId,
      experimentMap,
      permissions,
      selectAll,
      fetchFilters,
    ],
  );

  const submitBatchAction = useCallback(
    async (action: Action) => {
      try {
        await sendBatchActions(action);
        // if (action === Action.OpenTensorBoard && result) {
        //   openCommandResponse(result as CommandResponse);
        // }

        /*
         * Deselect selected rows since their states may have changed where they
         * are no longer part of the filter criteria.
         */
        setSelectedExperimentIds([]);
        setSelectAll(false);

        // Refetch experiment list to get updates based on batch action.
        await fetchExperiments();
      } catch (e) {
        const publicSubject =
          action === Action.OpenTensorBoard
            ? 'Unable to View TensorBoard for Selected Experiments'
            : `Unable to ${action} Selected Experiments`;
        handleError(e, {
          isUserTriggered: true,
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject,
          silent: false,
        });
      }
    },
    [fetchExperiments, sendBatchActions],
  );

  const showConfirmation = useCallback(
    (action: Action) => {
      modal.confirm({
        content: `
        Are you sure you want to ${action.toLocaleLowerCase()}
        all the eligible selected experiments?
      `,
        icon: <ExclamationCircleOutlined />,
        okText: /cancel/i.test(action) ? 'Confirm' : action,
        onOk: () => submitBatchAction(action),
        title: 'Confirm Batch Action',
      });
    },
    [submitBatchAction],
  );

  const handleBatchAction = useCallback(
    (action?: string) => {
      if (action === Action.OpenTensorBoard || action === Action.Move) {
        submitBatchAction(action);
      } else {
        showConfirmation(action as Action);
      }
    },
    [submitBatchAction, showConfirmation],
  );

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
          <>
            <TableActionBar
              actions={batchActions.map((action) => ({
                disabled: !availableBatchActions.includes(action),
                label: action,
              }))}
              selectAll={selectAll}
              selectedRowCount={selectedExperimentIds.length}
              onAction={handleBatchAction}
            />
            <GlideTable
              colorMap={colorMap}
              data={experiments}
              fetchExperiments={fetchExperiments}
              handleScroll={handleScroll}
              initialScrollPositionSet={initialScrollPositionSet}
              page={page}
              project={project}
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
      {modalExperimentMoveContextHolder}
    </Page>
  );
};

export default F_ExperimentList;
