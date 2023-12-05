import { Button, Space, Typography } from 'antd';
import Glossary, { InfoRow } from 'hew/Glossary';
import Icon from 'hew/Icon';
import { useModal } from 'hew/Modal';
import Spinner from 'hew/Spinner';
import Tags from 'hew/Tags';
import { useTheme } from 'hew/Theme';
import Tooltip from 'hew/Tooltip';
import useConfirm from 'hew/useConfirm';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge from 'components/Badge';
import ExperimentCreateModalComponent, {
  CreateExperimentType,
} from 'components/ExperimentCreateModal';
import ExperimentDeleteModalComponent from 'components/ExperimentDeleteModal';
import ExperimentEditModalComponent from 'components/ExperimentEditModal';
import ExperimentIcons from 'components/ExperimentIcons';
import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import ExperimentStopModalComponent from 'components/ExperimentStopModal';
import Link from 'components/Link';
import PageHeaderFoldable, { Option } from 'components/PageHeaderFoldable';
import TimeAgo from 'components/TimeAgo';
import TimeDuration from 'components/TimeDuration';
import { UNMANAGED_MESSAGE } from 'constant';
import { pausableRunStates, stateToLabel, terminalRunStates } from 'constants/states';
import useExperimentTags from 'hooks/useExperimentTags';
import useModalHyperparameterSearch from 'hooks/useModal/HyperparameterSearch/useModalHyperparameterSearch';
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
  ExperimentBase,
  JobState,
  RunState,
  TrialItem,
} from 'types';
import { getStateColorThemeVar } from 'utils/color';
import { getDuration } from 'utils/datetime';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import {
  canActionExperiment,
  getActionsForExperiment,
  isSingleTrialExperiment,
} from 'utils/experiment';
import { pluralizer } from 'utils/string';
import { openCommandResponse } from 'utils/wait';

import css from './ExperimentDetailsHeader.module.scss';

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
  Action.OpenTensorBoard,
  Action.HyperparameterSearch,
  Action.DownloadCode,
  Action.Edit,
  Action.Archive,
  Action.Unarchive,
  Action.Delete,
];

const ExperimentDetailsHeader: React.FC<Props> = ({
  experiment,
  fetchExperimentDetails,
  trial,
}: Props) => {
  const { getThemeVar } = useTheme();
  const [isChangingState, setIsChangingState] = useState(false);
  const [isRunningArchive, setIsRunningArchive] = useState<boolean>(false);
  const [isRunningTensorBoard, setIsRunningTensorBoard] = useState<boolean>(false);
  const [isRunningUnarchive, setIsRunningUnarchive] = useState<boolean>(false);
  const [isRunningDelete, setIsRunningDelete] = useState<boolean>(
    experiment.state === RunState.Deleting,
  );
  const [erroredTrialCount, setErroredTrialCount] = useState<number>();
  const [canceler] = useState(new AbortController());
  const confirm = useConfirm();
  const classes = [css.state];

  const maxRestarts = experiment.config.maxRestarts;
  const autoRestarts = trial?.autoRestarts ?? 0;

  const isPausable = pausableRunStates.has(experiment.state);
  const isPaused = experiment.state === RunState.Paused;
  const isTerminated = terminalRunStates.has(experiment.state);

  if (isTerminated) classes.push(css.terminated);

  const experimentTags = useExperimentTags(fetchExperimentDetails);

  const handleModalClose = useCallback(
    async () => await fetchExperimentDetails(),
    [fetchExperimentDetails],
  );

  const expPermissions = usePermissions();
  const isMovable =
    canActionExperiment(Action.Move, experiment) &&
    expPermissions.canMoveExperiment({ experiment });
  const canPausePlay = expPermissions.canModifyExperiment({
    workspace: { id: experiment.workspaceId },
  });

  const ExperimentStopModal = useModal(ExperimentStopModalComponent);
  const ExperimentMoveModal = useModal(ExperimentMoveModalComponent);
  const ExperimentDeleteModal = useModal(ExperimentDeleteModalComponent);
  const ContinueTrialModal = useModal(ExperimentCreateModalComponent);
  const ForkModal = useModal(ExperimentCreateModalComponent);
  const ExperimentEditModal = useModal(ExperimentEditModalComponent);

  const {
    contextHolder: modalHyperparameterSearchContextHolder,
    modalOpen: openModalHyperparameterSearch,
  } = useModalHyperparameterSearch({ experiment });

  const stateStyle = useMemo(
    () => ({
      backgroundColor: getThemeVar(getStateColorThemeVar(experiment.state)),
      color: getThemeVar(
        getStateColorThemeVar(experiment.state, { isOn: true, strongWeak: 'strong' }),
      ),
    }),
    [experiment.state, getThemeVar],
  );

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
        publicSubject: 'Unable to pause experiment.',
        silent: false,
        type: ErrorType.Server,
      });
    } finally {
      setIsChangingState(false);
    }
  }, [experiment.id, fetchExperimentDetails]);

  const handlePlayClick = useCallback(async () => {
    setIsChangingState(true);
    try {
      await activateExperiment({ experimentId: experiment.id });
      await fetchExperimentDetails();
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to activate experiment.',
        silent: false,
        type: ErrorType.Server,
      });
    } finally {
      setIsChangingState(false);
    }
  }, [experiment.id, fetchExperimentDetails]);

  const handleHyperparameterSearch = useCallback(() => {
    openModalHyperparameterSearch();
  }, [openModalHyperparameterSearch]);

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

  useEffect(() => {
    fetchErroredTrial();
  }, [fetchErroredTrial]);

  const headerOptions = useMemo(() => {
    const options: Partial<Record<Action, Option>> = {
      [Action.Unarchive]: {
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
      [Action.ContinueTrial]: {
        disabled: experiment.unmanaged,
        key: 'continue-trial',
        label: experiment.unmanaged ? (
          <Tooltip content={UNMANAGED_MESSAGE}>Continue Trial</Tooltip>
        ) : (
          'Continue Trial'
        ),
        onClick: ContinueTrialModal.open,
      },
      [Action.Delete]: {
        isLoading: isRunningDelete,
        key: 'delete',
        label: 'Delete',
        onClick: ExperimentDeleteModal.open,
      },
      [Action.HyperparameterSearch]: {
        disabled: experiment.unmanaged,
        key: 'hyperparameter-search',
        label: experiment.unmanaged ? (
          <Tooltip content={UNMANAGED_MESSAGE}>Hyperparameter Search</Tooltip>
        ) : (
          'Hyperparameter Search'
        ),
        onClick: handleHyperparameterSearch,
      },
      [Action.DownloadCode]: {
        icon: <Icon name="download" size="small" title={Action.DownloadCode} />,
        key: 'download-model',
        label: 'Download Experiment Code',
        onClick: (e) => {
          handlePath(e, { external: true, path: paths.experimentModelDef(experiment.id) });
        },
      },
      [Action.Retry]: {
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
                    'trial',
                  )} from their last available ${pluralizer(erroredTrialCount, 'checkpoint')}.`
                : 'Retry will resume the experiment from where it left off. Any previous progress will be retained.',
            okText: 'Retry',
            onConfirm: async () => {
              await continueExperiment({ id: experiment.id });
              await fetchExperimentDetails();
            },
            onError: handleError,
            title: 'Retry Experiment',
          });
        },
      },
      [Action.Fork]: {
        disabled: experiment.unmanaged,
        icon: <Icon name="fork" size="small" title={Action.Fork} />,
        key: 'fork',
        label: experiment.unmanaged ? <Tooltip content={UNMANAGED_MESSAGE}>Fork</Tooltip> : 'Fork',
        onClick: ForkModal.open,
      },
      [Action.Edit]: {
        key: 'edit',
        label: 'Edit',
        onClick: ExperimentEditModal.open,
      },
      [Action.Move]: {
        key: 'move',
        label: 'Move',
        onClick: ExperimentMoveModal.open,
      },
      [Action.OpenTensorBoard]: {
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
      [Action.Archive]: {
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
    };

    const availableActions = getActionsForExperiment(
      experiment,
      headerActions,
      expPermissions,
      erroredTrialCount,
    );

    return availableActions.map((action) => options[action]) as Option[];
  }, [
    expPermissions,
    isRunningArchive,
    ContinueTrialModal,
    isRunningDelete,
    ExperimentDeleteModal,
    handleHyperparameterSearch,
    ForkModal,
    ExperimentEditModal,
    ExperimentMoveModal,
    isRunningTensorBoard,
    isRunningUnarchive,
    experiment,
    fetchExperimentDetails,
    confirm,
    erroredTrialCount,
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
              size="large"
              state={experiment.state}
            />
            <div className={css.icon}>{iconNode}</div>
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
        value: (
          <Typography.Paragraph
            disabled={!experiment.description}
            ellipsis={{ rows: 1, tooltip: true }}
            style={{ margin: 0 }}>
            {experiment.description || 'N/A'}
          </Typography.Paragraph>
        ),
      },
    ];
    if (experiment.forkedFrom && experiment.config.searcher.sourceTrialId) {
      rows.push({
        label: 'Continued from',
        value: (
          <Link
            className={css.link}
            path={paths.trialDetails(experiment.config.searcher.sourceTrialId)}>
            Trial {experiment.config.searcher.sourceTrialId}
          </Link>
        ),
      });
    }
    if (experiment.forkedFrom && !experiment.config.searcher.sourceTrialId) {
      rows.push({
        label: 'Forked from',
        value: (
          <Link className={css.link} path={paths.experimentDetails(experiment.forkedFrom)}>
            Experiment {experiment.forkedFrom}
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
          <Link className={css.link} path={paths.jobs()}>
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
  }, [autoRestarts, disabled, experiment, experimentTags, jobInfoLinkText, maxRestarts]);

  return (
    <>
      <PageHeaderFoldable
        foldableContent={<Glossary content={foldableRows} />}
        leftContent={
          <Space align="center" className={css.base}>
            <Spinner spinning={isChangingState}>
              <div className={css.stateIcon}>
                {isActionableIcon(experiment.state) ? (
                  <div className={classes.join(' ')} style={stateStyle}>
                    {isPausable && (
                      <Button
                        className={
                          isShownAnimation(experiment.state)
                            ? css.buttonWithAnimation
                            : css.buttonPause
                        }
                        disabled={!canPausePlay}
                        icon={returnStatusIcon(<Icon name="pause" size="large" title="Pause" />)}
                        shape="circle"
                        onClick={handlePauseClick}
                      />
                    )}
                    {isPaused && (
                      <Button
                        className={
                          isShownAnimation(experiment.state)
                            ? css.buttonWithAnimation
                            : css.buttonPlay
                        }
                        disabled={!canPausePlay}
                        icon={returnStatusIcon(<Icon name="play" size="large" title="Play" />)}
                        shape="circle"
                        onClick={handlePlayClick}
                      />
                    )}
                    {!isTerminated && (
                      <Button
                        className={css.buttonStop}
                        disabled={!canPausePlay}
                        icon={<Icon name="stop" size="large" title="Stop" />}
                        shape="circle"
                        onClick={ExperimentStopModal.open}
                      />
                    )}
                    <label>{stateToLabel(experiment.state)}</label>
                  </div>
                ) : (
                  <ExperimentIcons state={experiment.state} />
                )}
              </div>
            </Spinner>
            <div className={css.id}>Experiment {experiment.id}</div>
            <div className={css.name} role="experimentName">
              {experiment.name}
            </div>
            {experiment.unmanaged && (
              <Badge tooltip="Workload not managed by Determined" type="Header">
                Unmanaged
              </Badge>
            )}
            {trial ? (
              <>
                <Icon name="arrow-right" size="tiny" title="Trial" />
                <div className={css.trial}>Trial {trial.id}</div>
              </>
            ) : null}
          </Space>
        }
        options={headerOptions}
      />
      <ExperimentHeaderProgress experiment={experiment} />
      <ContinueTrialModal.Component
        experiment={experiment}
        trial={trial}
        type={CreateExperimentType.ContinueTrial}
      />
      <ForkModal.Component experiment={experiment} type={CreateExperimentType.Fork} />
      <ExperimentDeleteModal.Component experiment={experiment} />
      <ExperimentMoveModal.Component
        experimentIds={isMovable ? [experiment.id] : []}
        sourceProjectId={experiment.projectId}
        sourceWorkspaceId={experiment.workspaceId}
        onSubmit={handleModalClose}
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
      {modalHyperparameterSearchContextHolder}
    </>
  );
};

export default ExperimentDetailsHeader;
