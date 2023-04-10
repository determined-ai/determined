import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Rectangle } from '@glideapps/glide-data-grid';
import { Row } from 'antd';
import SkeletonButton from 'antd/es/skeleton/Button';
import useModalExperimentMove from 'hooks/useModal/Experiment/useModalExperimentMove';
import { observable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import { useModal } from 'components/kit/Modal';
import Page from 'components/Page';
import usePermissions from 'hooks/usePermissions';
import useResize from 'hooks/useResize';
import {
  activateExperiments,
  archiveExperiments,
  cancelExperiments,
  deleteExperiments,
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
import { openCommandResponse } from 'utils/wait';

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
] as const;

type BatchAction = (typeof batchActions)[number];

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
  const [clearSelectionTrigger, setClearSelectionTrigger] = useState(0);
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

  const unselect = useCallback(() => {
    setClearSelectionTrigger((prev) => prev + 1);
    setSelectAll(false);
  }, []);

  const handleModalClose = useCallback(() => {
    unselect();
    fetchExperiments();
  }, [fetchExperiments, unselect]);

  const ExperimentMoveModal = useModal(ExperimentMoveModalComponent);

  const experimentMap = useMemo(() => {
    return experiments.filter(Loadable.isLoaded).reduce((acc, experiment) => {
      acc[experiment.data.id] = getProjectExperimentForExperimentItem(experiment.data, project);
      return acc;
    }, {} as Record<RecordKey, ProjectExperiment>);
  }, [experiments, project]);

  const availableBatchActions = useMemo(() => {
    if (selectAll) return batchActions;
    const experiments = selectedExperimentIds.map((id) => experimentMap[id]) ?? [];
    return getActionsForExperimentsUnion(experiments, [...batchActions], permissions);
    // Spreading batchActions is so TypeScript doesn't complain that it's readonly.
  }, [experimentMap, permissions, selectAll, selectedExperimentIds]);

  const sendBatchActions = useCallback(
    async (action: BatchAction): Promise<BulkActionError[] | void> => {
      switch (action) {
        case Action.OpenTensorBoard:
          return openCommandResponse(
            await openOrCreateTensorBoard({
              experimentIds: selectedExperimentIds,
              filters: selectAll ? fetchFilters : undefined,
              workspaceId: project?.workspaceId,
            }),
          );
        case Action.Move:
          return ExperimentMoveModal.open();
        case Action.Activate:
          return await activateExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? fetchFilters : undefined,
          });
        case Action.Archive:
          return await archiveExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? fetchFilters : undefined,
          });
        case Action.Cancel:
          return await cancelExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? fetchFilters : undefined,
          });
        case Action.Kill:
          return await killExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? fetchFilters : undefined,
          });
        case Action.Pause:
          return await pauseExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? fetchFilters : undefined,
          });
        case Action.Unarchive:
          return await unarchiveExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? fetchFilters : undefined,
          });
        case Action.Delete:
          return await deleteExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? fetchFilters : undefined,
          });
      }
    },
    [selectedExperimentIds, selectAll, fetchFilters, project?.workspaceId, ExperimentMoveModal],
  );

  const submitBatchAction = useCallback(
    async (action: BatchAction) => {
      try {
        await sendBatchActions(action);

        /*
         * Deselect selected rows since their states may have changed where they
         * are no longer part of the filter criteria.
         */
        unselect();

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
    [fetchExperiments, sendBatchActions, unselect],
  );

  const showConfirmation = useCallback(
    (action: BatchAction) => {
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
    (action: string) => {
      if (action === Action.OpenTensorBoard) {
        submitBatchAction(action);
      } else if (action === Action.Move) {
        sendBatchActions(action);
      } else {
        showConfirmation(action as BatchAction);
      }
    },
    [submitBatchAction, sendBatchActions, showConfirmation],
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
              clearSelectionTrigger={clearSelectionTrigger}
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
      <ExperimentMoveModal.Component
        experimentIds={selectedExperimentIds.filter(
          (id) =>
            canActionExperiment(Action.Move, experimentMap[id]) &&
            permissions.canMoveExperiment({ experiment: experimentMap[id] }),
        )}
        filters={selectAll ? fetchFilters : undefined}
        sourceProjectId={project?.id}
        sourceWorkspaceId={project?.workspaceId}
        onClose={handleModalClose}
      />
    </Page>
  );
};

export default F_ExperimentList;
