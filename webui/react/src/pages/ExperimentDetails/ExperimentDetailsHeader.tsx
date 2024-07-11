import Button, { status } from 'hew/Button';
import Column from 'hew/Column';
import Dropdown from 'hew/Dropdown';
import Glossary, { InfoRow } from 'hew/Glossary';
import Icon from 'hew/Icon';
import { useModal } from 'hew/Modal';
import Row from 'hew/Row';
import Spinner from 'hew/Spinner';
import Tags from 'hew/Tags';
import { stateColorMapping } from 'hew/Theme';
import Tooltip from 'hew/Tooltip';
import { Body } from 'hew/Typography';
import useConfirm from 'hew/useConfirm';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge from 'components/Badge';
import ExperimentContinueModalComponent, {
  ContinueExperimentType,
} from 'components/ExperimentContinueModal';
import ExperimentCreateModalComponent, {
  CreateExperimentType,
} from 'components/ExperimentCreateModal';
import ExperimentDeleteModalComponent from 'components/ExperimentDeleteModal';
import ExperimentEditModalComponent from 'components/ExperimentEditModal';
import ExperimentIcons from 'components/ExperimentIcons';
import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import ExperimentRetainLogsModalComponent from 'components/ExperimentRetainLogsModal';
import ExperimentStopModalComponent from 'components/ExperimentStopModal';
import HyperparameterSearchModalComponent from 'components/HyperparameterSearchModal';
import Link from 'components/Link';
import PageHeaderFoldable, { Option, renderOptionLabel } from 'components/PageHeaderFoldable';
import TimeAgo from 'components/TimeAgo';
import TimeDuration from 'components/TimeDuration';
import { UNMANAGED_MESSAGE } from 'constant';
import { pausableRunStates, stateToLabel, terminalRunStates } from 'constants/states';
import useExperimentTags from 'hooks/useExperimentTags';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import ExperimentHeaderProgress from 'pages/ExperimentDetails/Header/ExperimentHeaderProgress';
import { handlePath, paths } from 'routes/utils';
import {
  activateExperiment,
  archiveExperiment,
  continueExperiment,
  getExpTrials,
  openOrCreateTensorBoard,
  pauseExperiment,
  unarchiveExperiment,
} from 'services/api';
import { Experimentv1State } from 'services/api-ts-sdk';
import {
  ExperimentAction as Action,
  CompoundRunState,
  ContinuableNonSingleSearcherName,
  ExperimentBase,
  JobState,
  RunState,
  TrialItem,
} from 'types';
import { getDuration } from 'utils/datetime';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import {
  canActionExperiment,
  getActionsForExperiment,
  isSingleTrialExperiment,
} from 'utils/experiment';
import { routeToReactUrl } from 'utils/routes';
import { capitalize, pluralizer } from 'utils/string';
import { openCommandResponse } from 'utils/wait';

import css from './ExperimentDetailsHeader.module.scss';

export interface ActionOptions {
  content?: React.ReactNode;
  key: string;
  menuOptions: Option[];
}
// Actionable means that user can take an action, such as pause, stop
const isActionableIcon = (state: CompoundRunState): boolean => {
  switch (state) {
    case JobState.SCHEDULED:
    case JobState.SCHEDULEDBACKFILLED:
    case JobState.QUEUED:
    case RunState.Queued:
    case RunState.Starting:
    case RunState.Pulling:
    case RunState.Running:
    case RunState.Paused:
    case RunState.Active:
    case RunState.Unspecified:
    case JobState.UNSPECIFIED:
      return true;
    case RunState.Completed:
    case RunState.Error:
    case RunState.Deleted:
    case RunState.Deleting:
    case RunState.DeleteFailed:
      return false;
    default:
      return false;
  }
};

// If status(state) icon has actionable butotn(s) and animation fits the design,
// show  animation around the icon
const isShownAnimation = (state: CompoundRunState): boolean => {
  switch (state) {
    case JobState.SCHEDULED:
    case JobState.SCHEDULEDBACKFILLED:
    case JobState.QUEUED:
    case RunState.Queued:
    case RunState.Starting:
    case RunState.Pulling:
    case RunState.Running:
      return true;
    case RunState.Active:
    case RunState.Paused:
    case RunState.Unspecified:
    case JobState.UNSPECIFIED:
    default:
      return false;
  }
};

interface Props {
  experiment: ExperimentBase;
  fetchExperimentDetails: () => Promise<void>;
  name?: string;
  trial?: TrialItem;
  // TODO: separate components for
  // 1) displaying an abbreviated string as an Avatar and
  // 2) finding user by userId in the store and displaying string Avatar or profile image
  userId?: number;
}

const headerActions = [
  Action.Retry,
  Action.Fork,
  Action.ContinueTrial,
  Action.Move,
  Action.RetainLogs,
  Action.OpenTensorBoard,
  Action.HyperparameterSearch,
  Action.DownloadCode,
  Action.Edit,
  Action.Archive,
  Action.Unarchive,
  Action.Delete,
];

const ExperimentEntityCopyMap = {
  experiment: 'experiment',
  trial: 'trial',
};

const RunEntityCopyMap = {
  experiment: 'search',
  trial: 'run',
};

// prettier-ignore
const ExperimentDetailsHeader: React.FC<Props> = ({
  experiment,
  fetchExperimentDetails,
  trial,
}: Props) => {
  const [isChangingState, setIsChangingState] = useState(false);
  const [isRunningArchive, setIsRunningArchive] = useState<boolean>(false);
  const [isRunningTensorBoard, setIsRunningTensorBoard] = useState<boolean>(false);
  const [isRunningUnarchive, setIsRunningUnarchive] = useState<boolean>(false);
  const [isRunningDelete, setIsRunningDelete] = useState<boolean>(
    experiment.state === RunState.Deleting,
  );
  const [isRunningContinue, setIsRunningContinue] = useState<boolean>(false);
  const [erroredTrialCount, setErroredTrialCount] = useState<number>();
  const [canceler] = useState(new AbortController());
  const confirm = useConfirm();
  const f_flat_runs = useFeature().isOn('flat_runs');
  const copyMap = f_flat_runs ? RunEntityCopyMap : ExperimentEntityCopyMap;

  const maxRestarts = experiment.config.maxRestarts;
  const autoRestarts = trial?.autoRestarts ?? 0;

  const isPausable = pausableRunStates.has(experiment.state);
  const isPaused = experiment.state === RunState.Paused;
  const isTerminated = terminalRunStates.has(experiment.state);

  const experimentTags = useExperimentTags(fetchExperimentDetails);

  const handleModalClose = useCallback(
    async () => await fetchExperimentDetails(),
    [fetchExperimentDetails],
  );

  const expPermissions = usePermissions();
  const isMovable =
    canActionExperiment(Action.Move, experiment) &&
    expPermissions.canMoveExperiment({ experiment });
  const canModifyExp = canActionExperiment(Action.Move, experiment) &&
    expPermissions.canModifyExperiment({
      workspace: { id: experiment.workspaceId },
    });
  const canPausePlay = expPermissions.canModifyExperiment({
    workspace: { id: experiment.workspaceId },
  });

  const ExperimentStopModal = useModal(ExperimentStopModalComponent);
  const ExperimentMoveModal = useModal(ExperimentMoveModalComponent);
  const ExperimentRetainLogsModal = useModal(ExperimentRetainLogsModalComponent);
  const ExperimentDeleteModal = useModal(ExperimentDeleteModalComponent);
  const ReactivateExperimentModal = useModal(ExperimentContinueModalComponent);
  const ContinueExperimentModal = useModal(ExperimentContinueModalComponent);
  const ForkModal = useModal(ExperimentCreateModalComponent);
  const ExperimentEditModal = useModal(ExperimentEditModalComponent);
  const ContinueTrialModal = useModal(ExperimentCreateModalComponent);
  const HyperparameterSearchModal = useModal(HyperparameterSearchModalComponent);

  const disabled =
    experiment?.parentArchived ||
    experiment?.archived ||
    !expPermissions.canModifyExperimentMetadata({ workspace: { id: experiment?.workspaceId } });

  const handlePauseClick = useCallback(async () => {
    setIsChangingState(true);
    try {
      await pauseExperiment({ experimentId: experiment.id });
      await fetchExperimentDetails();
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: `Unable to pause ${copyMap.experiment}.`,
        silent: false,
        type: ErrorType.Server,
      });
    } finally {
      setIsChangingState(false);
    }
  }, [copyMap, experiment.id, fetchExperimentDetails]);

  const handlePlayClick = useCallback(async () => {
    setIsChangingState(true);
    try {
      await activateExperiment({ experimentId: experiment.id });
      await fetchExperimentDetails();
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: `Unable to activate ${copyMap.experiment}.`,
        silent: false,
        type: ErrorType.Server,
      });
    } finally {
      setIsChangingState(false);
    }
  }, [copyMap, experiment.id, fetchExperimentDetails]);

  const fetchErroredTrial = useCallback(async () => {
    // No need to fetch errored trial count if it's single trial experiment or experiment is not completed.
    if (isSingleTrialExperiment(experiment) || experiment.state !== RunState.Completed) return;
    const res = await getExpTrials(
      {
        id: experiment.id,
        limit: 1,
        states: [Experimentv1State.ERROR],
      },
      { signal: canceler.signal },
    );
    setErroredTrialCount(res.pagination.total);
  }, [experiment, canceler]);

  useEffect(() => {
    setIsRunningArchive(false);
    setIsRunningUnarchive(false);
  }, [experiment.archived]);

  useEffect(() => {
    setIsRunningDelete(experiment.state === RunState.Deleting);
  }, [experiment.state]);

  const onClickContinueMultiTrialExp = useCallback(async () => {
    try {
      setIsRunningContinue(true);
    await continueExperiment({
      id: experiment.id,
    });
    const newPath = paths.experimentDetails(experiment.id);
    routeToReactUrl(paths.reload(newPath));
  } catch (e) {
    handleError(e, {
      level: ErrorLevel.Error,
      publicMessage: 'Please try again later.',
      publicSubject: 'Unable to continue this experiment.',
      silent: false,
      type: ErrorType.Server,
    });
  } finally {
    setIsRunningContinue(false);
  }
  }, [experiment.id]);

  const continueExperimentOption = useMemo(
    () =>
      experiment?.config.searcher.name === 'single'
        ? {
          content: (
            <Dropdown
              menu={[
                {
                  key: 'Create New Experiment',
                  label: f_flat_runs
                    ? 'Create New Run'
                    : 'Create New Experiment...',
                },
                {
                  key: 'Reactivate Current Trial',
                  label: `Reactivate Current ${capitalize(copyMap.trial)}...`,
                },
              ]}
              onClick={(key: string) => {
                if (key === 'Create New Experiment') ContinueExperimentModal.open();
                if (key === 'Reactivate Current Trial') ReactivateExperimentModal.open();
              }}>
              <Button disabled={experiment.unmanaged}>Continue {capitalize(copyMap.trial)}</Button>
            </Dropdown>
          ),
          menuOptions: [
            {
              key: 'create-new-experiment',
              label: experiment.unmanaged ? (
                <Tooltip content={UNMANAGED_MESSAGE}>Continue {capitalize(copyMap.trial)}</Tooltip>
              ) : f_flat_runs ? 'Create New Run' : 'Create New Experiment'
              ,
              onClick: ContinueExperimentModal.open,
            },
            {
              key: 'reactivate-current-trial',
              label: experiment.unmanaged ? (
                <Tooltip content={UNMANAGED_MESSAGE}>Reactivate Current {capitalize(copyMap.trial)}</Tooltip>
              ) : (
                `Reactivate Current ${capitalize(copyMap.trial)}`
              ),
              onClick: ReactivateExperimentModal.open,
            },
          ],
        }
        : {
          menuOptions: [
            {
              disabled: experiment.unmanaged,
              isLoading: isRunningContinue,
              key: 'continue-trial',
              label: experiment.unmanaged ? (
                <Tooltip content={UNMANAGED_MESSAGE}>Continue {capitalize(copyMap.trial)}</Tooltip>
              ) : (
                `Continue ${capitalize(isSingleTrialExperiment(experiment) ? copyMap.trial : copyMap.experiment)}`
              ),
              onClick: ContinuableNonSingleSearcherName.has(experiment.config.searcher.name) ? onClickContinueMultiTrialExp : ContinueTrialModal.open,
            },
          ],
        },
    [copyMap, experiment, ContinueExperimentModal, f_flat_runs, ReactivateExperimentModal, isRunningContinue, onClickContinueMultiTrialExp, ContinueTrialModal.open],
  );

  useEffect(() => {
    fetchErroredTrial();
  }, [fetchErroredTrial]);

  const headerOptions = useMemo(() => {
    const options: Partial<Record<Action, ActionOptions>> = {
      [Action.Unarchive]: {
        key: 'unarchive',
        menuOptions: [
          {
            isLoading: isRunningArchive,
            key: 'unarchive',
            label: 'Unarchive',
            onClick: async (): Promise<void> => {
              setIsRunningUnarchive(true);
              try {
                await unarchiveExperiment({ experimentId: experiment.id });
                await fetchExperimentDetails();
              } catch (e) {
                setIsRunningUnarchive(false);
              }
            },
          },
        ],
      },
      [Action.ContinueTrial]: {
        ...continueExperimentOption,
        key: 'continue-trial',
      },
      [Action.Delete]: {
        key: 'delete',
        menuOptions: [
          {
            isLoading: isRunningDelete,
            key: 'delete',
            label: 'Delete',
            onClick: ExperimentDeleteModal.open,
          },
        ],
      },
      [Action.HyperparameterSearch]: {
        key: 'hyperparameter-search',
        menuOptions: [
          {
            disabled: experiment.unmanaged,
            key: 'hyperparameter-search',
            label: experiment.unmanaged ? (
              <Tooltip content={UNMANAGED_MESSAGE}>Hyperparameter Search</Tooltip>
            ) : (
              'Hyperparameter Search'
            ),
            onClick: HyperparameterSearchModal.open,
          },
        ],
      },
      [Action.DownloadCode]: {
        key: 'download-model',
        menuOptions: [
          {
            icon: <Icon name="download" size="small" title={Action.DownloadCode} />,
            key: 'download-model',
            label: 'Download Experiment Code',
            onClick: (e) => {
              handlePath(e, { external: true, path: paths.experimentModelDef(experiment.id) });
            },
          },
        ],
      },
      [Action.Retry]: {
        key: 'retry',
        menuOptions: [
          {
            disabled: experiment.unmanaged,
            icon: <Icon decorative name="reset" />,
            key: 'retry',
            label: erroredTrialCount ?? 0 > 0 ? `Retry Errored (${erroredTrialCount})` : 'Retry',
            onClick: () => {
              confirm({
                content:
                  erroredTrialCount && erroredTrialCount > 0
                    ? `Retry will attempt to complete ${erroredTrialCount} errored ${pluralizer(
                      erroredTrialCount,
                      copyMap.trial,
                    )} from their last available ${pluralizer(erroredTrialCount, 'checkpoint')}.`
                    : `Retry will resume the ${copyMap.experiment} from where it left off. Any previous progress will be retained.`,
                okText: 'Retry',
                onConfirm: async () => {
                  await continueExperiment({ id: experiment.id });
                  await fetchExperimentDetails();
                },
                onError: handleError,
                title: `Retry ${capitalize(copyMap.experiment)}`,
              });
            },
          },
        ],
      },
      [Action.Fork]: {
        key: 'fork',
        menuOptions: [
          {
            disabled: experiment.unmanaged,
            icon: <Icon name="fork" size="small" title={Action.Fork} />,
            key: 'fork',
            label: experiment.unmanaged ? (
              <Tooltip content={UNMANAGED_MESSAGE}>Fork</Tooltip>
            ) : (
              'Fork'
            ),
            onClick: ForkModal.open,
          },
        ],
      },
      [Action.Edit]: {
        key: 'edit',
        menuOptions: [
          {
            key: 'edit',
            label: 'Edit',
            onClick: ExperimentEditModal.open,
          },
        ],
      },
      [Action.Move]: {
        key: 'move',
        menuOptions: [
          {
            key: 'move',
            label: 'Move',
            onClick: ExperimentMoveModal.open,
          },
        ],
      },
      [Action.RetainLogs]: {
        key: 'retain-logs',
        menuOptions: [
          {
            key: 'retain-logs',
            label: 'Retain Logs',
            onClick: ExperimentRetainLogsModal.open,
          },
        ],
      },
      [Action.OpenTensorBoard]: {
        key: 'tensorboard',
        menuOptions: [
          {
            disabled: experiment.unmanaged,
            icon: <Icon name="tensor-board" size="small" title={Action.OpenTensorBoard} />,
            isLoading: isRunningTensorBoard,
            key: 'tensorboard',
            label: experiment.unmanaged ? (
              <Tooltip content={UNMANAGED_MESSAGE}>TensorBoard</Tooltip>
            ) : (
              'TensorBoard'
            ),
            onClick: async () => {
              setIsRunningTensorBoard(true);
              try {
                const commandResponse = await openOrCreateTensorBoard({
                  experimentIds: [experiment.id],
                  workspaceId: experiment.workspaceId,
                });
                openCommandResponse(commandResponse);
                setIsRunningTensorBoard(false);
              } catch (e) {
                setIsRunningTensorBoard(false);
              }
            },
          },
        ],
      },
      [Action.Archive]: {
        key: 'archive',
        menuOptions: [
          {
            isLoading: isRunningUnarchive,
            key: 'archive',
            label: 'Archive',
            onClick: async (): Promise<void> => {
              setIsRunningArchive(true);
              try {
                await archiveExperiment({ experimentId: experiment.id });
                await fetchExperimentDetails();
              } catch (e) {
                setIsRunningArchive(false);
              }
            },
          },
        ],
      },
    };

    const availableActions = getActionsForExperiment(
      experiment,
      headerActions,
      expPermissions,
      erroredTrialCount,
    );

    return availableActions.map((action) => options[action]) as ActionOptions[];
  }, [
    copyMap,
    isRunningArchive,
    continueExperimentOption,
    isRunningDelete,
    ExperimentDeleteModal.open,
    experiment,
    HyperparameterSearchModal.open,
    erroredTrialCount,
    ForkModal.open,
    ExperimentEditModal.open,
    ExperimentMoveModal.open,
    ExperimentRetainLogsModal.open,
    isRunningTensorBoard,
    isRunningUnarchive,
    expPermissions,
    fetchExperimentDetails,
    confirm,
  ]);

  const jobInfoLinkText = useMemo(() => {
    if (!experiment.jobSummary) return 'Not available';
    const isJobOrderAvailable = experiment.jobSummary.jobsAhead >= 0;
    const isFirstJob = experiment.jobSummary.jobsAhead === 0;
    if (!isJobOrderAvailable) return 'Available here';
    if (isFirstJob) return 'No jobs ahead of this one';
    return `${experiment.jobSummary.jobsAhead} jobs ahead of this one`;
  }, [experiment.jobSummary]);

  const returnStatusIcon = useCallback(
    (iconNode: React.ReactNode): React.ReactNode => {
      {
        return isShownAnimation(experiment.state) ? (
          <>
            <ExperimentIcons
              backgroundColor="white" // only gets applied for scheduled and queued states
              opacity={0.25} // only gets applied for scheduled and queued states
              showTooltip={false}
              size="big"
              state={experiment.state}
            />
            <span className={css.backgroundIcon}>{iconNode}</span>
          </>
        ) : (
          iconNode
        );
      }
    },
    [experiment.state],
  );

  const foldableRows: InfoRow[] = useMemo(() => {
    const rows = [
      {
        label: 'Description',
        value: <Body truncate={{ rows: 1, tooltip: true }}>{experiment.description || 'N/A'}</Body>,
      },
    ];
    if (experiment.forkedFrom && experiment.config.searcher.sourceTrialId) {
      rows.push({
        label: 'Continued from',
        value: (
          <Link
            path={paths.trialDetails(experiment.config.searcher.sourceTrialId)}>
            {capitalize(copyMap.trial)} {experiment.config.searcher.sourceTrialId}
          </Link>
        ),
      });
    }
    if (experiment.forkedFrom && !experiment.config.searcher.sourceTrialId) {
      rows.push({
        label: 'Forked from',
        value: (
          <Link path={paths.experimentDetails(experiment.forkedFrom)}>
            {capitalize(copyMap.experiment)} {experiment.forkedFrom}
          </Link>
        ),
      });
    }
    rows.push({ label: 'Started', value: <TimeAgo datetime={experiment.startTime} long /> });
    if (experiment.endTime != null) {
      rows.push({
        label: 'Duration',
        value: <TimeDuration duration={getDuration(experiment)} />,
      });
    }
    if (experiment.jobSummary && !terminalRunStates.has(experiment.state)) {
      rows.push({
        label: 'Job info',
        value: (
          <Link path={paths.jobs()}>
            {jobInfoLinkText}
          </Link>
        ),
      });
    }
    rows.push({
      label: 'Auto restarts',
      value: (
        <div>
          {autoRestarts}
          {maxRestarts ? `/${maxRestarts}` : ''}
        </div>
      ),
    });
    rows.push({
      label: 'Tags',
      value: (
        <Tags
          disabled={disabled}
          ghost={true}
          tags={experiment.config.labels || []}
          onAction={experimentTags.handleTagListChange(
            experiment.id,
            experiment.config.labels || [],
          )}
        />
      ),
    });

    return rows;
  }, [autoRestarts, copyMap, disabled, experiment, experimentTags, jobInfoLinkText, maxRestarts]);

  return (
    <>
      <PageHeaderFoldable
        foldableContent={<Glossary content={foldableRows} />}
        leftContent={
          <Row align="center" wrap>
            <Column>
              <Spinner spinning={isChangingState}>
                {isActionableIcon(experiment.state) ? (
                  <Button shape="round" status={stateColorMapping[experiment.state] as status}>
                    {isPausable && (
                      <Button
                        disabled={!canPausePlay}
                        icon={returnStatusIcon(<Icon name="pause" size="large" title="Pause" />)}
                        shape="circle"
                        status={stateColorMapping[experiment.state] as status}
                        onClick={handlePauseClick}
                      />
                    )}
                    {isPaused && (
                      <Button
                        disabled={!canPausePlay}
                        icon={returnStatusIcon(<Icon name="play" size="large" title="Play" />)}
                        shape="circle"
                        status={stateColorMapping[experiment.state] as status}
                        onClick={handlePlayClick}
                      />
                    )}
                    {!isTerminated && (
                      <Button
                        disabled={!canPausePlay}
                        icon={<Icon name="stop" size="large" title="Stop" />}
                        shape="circle"
                        status={stateColorMapping[experiment.state] as status}
                        type="text"
                        onClick={ExperimentStopModal.open}
                      />
                    )}
                    <label className={css.buttonLabel}>{stateToLabel(experiment.state)}</label>
                  </Button>
                ) : (
                  <ExperimentIcons state={experiment.state} />
                )}
              </Spinner>
            </Column>
            <span>Experiment {experiment.id}</span>
            <span role="experimentName">
              {experiment.name}
            </span>
            {experiment.unmanaged && (
              <Badge tooltip="Workload not managed by Determined" type="Header">
                Unmanaged
              </Badge>
            )}
            {trial ? (
              <>
                <Icon name="arrow-right" size="tiny" title={capitalize(copyMap.trial)} />
                <span>{capitalize(copyMap.trial)} {trial.id}</span>
              </>
            ) : null}
          </Row>
        }
        options={headerOptions.map((option) => ({
          content: option?.content
            ? option.content
            : option.menuOptions.map((menuOption) => (
              <Button
                disabled={menuOption.disabled || !menuOption.onClick}
                icon={menuOption?.icon}
                key={menuOption.key}
                loading={menuOption.isLoading}
                onClick={menuOption.onClick}>
                {renderOptionLabel(menuOption)}
              </Button>
            )),
          key: option.key,
          menuOptions: option.menuOptions,
        }))}
      />
      <ExperimentHeaderProgress experiment={experiment} />
      <ForkModal.Component experiment={experiment} type={CreateExperimentType.Fork} />
      <ReactivateExperimentModal.Component
        experiment={experiment}
        type={ContinueExperimentType.Reactivate}
      />
      <ContinueExperimentModal.Component
        experiment={experiment}
        trial={trial}
        type={ContinueExperimentType.Continue}
      />
      <ContinueTrialModal.Component
        experiment={experiment}
        trial={trial}
        type={CreateExperimentType.ContinueTrial}
      />
      <ExperimentDeleteModal.Component experiment={experiment} />
      <ExperimentMoveModal.Component
        experimentIds={isMovable ? [experiment.id] : []}
        sourceProjectId={experiment.projectId}
        sourceWorkspaceId={experiment.workspaceId}
        onSubmit={handleModalClose}
      />
      <ExperimentRetainLogsModal.Component
        experimentIds={canModifyExp ? [experiment.id] : []}
        projectId={experiment.projectId}
      />
      <ExperimentStopModal.Component
        experimentId={experiment.id}
        onClose={fetchExperimentDetails}
      />
      <ExperimentEditModal.Component
        description={experiment.description ?? ''}
        experimentId={experiment.id}
        experimentName={experiment.name}
        onEditComplete={fetchExperimentDetails}
      />
      <HyperparameterSearchModal.Component
        closeModal={HyperparameterSearchModal.close}
        experiment={experiment}
      />
    </>
  );
};

export default ExperimentDetailsHeader;
