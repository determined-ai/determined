import { Button, Space, Typography } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import BreadcrumbBar from 'components/BreadcrumbBar';
import ExperimentCreateModalComponent, {
  CreateExperimentType,
} from 'components/ExperimentCreateModal';
import ExperimentDeleteModalComponent from 'components/ExperimentDeleteModal';
import ExperimentEditModalComponent from 'components/ExperimentEditModal';
import ExperimentIcons from 'components/ExperimentIcons';
import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import ExperimentStopModalComponent from 'components/ExperimentStopModal';
import InfoBox, { InfoRow } from 'components/InfoBox';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import Tags from 'components/kit/Tags';
import Link from 'components/Link';
import PageHeaderFoldable, { Option } from 'components/PageHeaderFoldable';
import TimeAgo from 'components/TimeAgo';
import TimeDuration from 'components/TimeDuration';
import { pausableRunStates, stateToLabel, terminalRunStates } from 'constants/states';
import useExperimentTags from 'hooks/useExperimentTags';
import useModalHyperparameterSearch from 'hooks/useModal/HyperparameterSearch/useModalHyperparameterSearch';
import usePermissions from 'hooks/usePermissions';
import ExperimentHeaderProgress from 'pages/ExperimentDetails/Header/ExperimentHeaderProgress';
import { handlePath, paths } from 'routes/utils';
import {
  activateExperiment,
  archiveExperiment,
  openOrCreateTensorBoard,
  pauseExperiment,
  unarchiveExperiment,
} from 'services/api';
import Spinner from 'shared/components/Spinner/Spinner';
import { getDuration } from 'shared/utils/datetime';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { getStateColorCssVar } from 'themes';
import {
  ExperimentAction as Action,
  CompoundRunState,
  ExperimentBase,
  JobState,
  RunState,
  TrialItem,
} from 'types';
import handleError from 'utils/error';
import { canActionExperiment, getActionsForExperiment } from 'utils/experiment';
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
      return false;
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
  const [isChangingState, setIsChangingState] = useState(false);
  const [isRunningArchive, setIsRunningArchive] = useState<boolean>(false);
  const [isRunningTensorBoard, setIsRunningTensorBoard] = useState<boolean>(false);
  const [isRunningUnarchive, setIsRunningUnarchive] = useState<boolean>(false);
  const [isRunningDelete, setIsRunningDelete] = useState<boolean>(
    experiment.state === RunState.Deleting,
  );
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
      backgroundColor: getStateColorCssVar(experiment.state),
      color: getStateColorCssVar(experiment.state, { isOn: true, strongWeak: 'strong' }),
    }),
    [experiment.state],
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

  useEffect(() => {
    setIsRunningArchive(false);
    setIsRunningUnarchive(false);
  }, [experiment.archived]);

  useEffect(() => {
    setIsRunningDelete(experiment.state === RunState.Deleting);
  }, [experiment.state]);

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
        key: 'continue-trial',
        label: 'Continue Trial',
        onClick: ContinueTrialModal.open,
      },
      [Action.Delete]: {
        isLoading: isRunningDelete,
        key: 'delete',
        label: 'Delete',
        onClick: ExperimentDeleteModal.open,
      },
      [Action.HyperparameterSearch]: {
        key: 'hyperparameter-search',
        label: 'Hyperparameter Search',
        onClick: handleHyperparameterSearch,
      },
      [Action.DownloadCode]: {
        icon: <Icon name="download" size="small" />,
        key: 'download-model',
        label: 'Download Experiment Code',
        onClick: (e) => {
          handlePath(e, { external: true, path: paths.experimentModelDef(experiment.id) });
        },
      },
      [Action.Fork]: {
        icon: <Icon name="fork" size="small" />,
        key: 'fork',
        label: 'Fork',
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
        icon: <Icon name="tensor-board" size="small" />,
        isLoading: isRunningTensorBoard,
        key: 'tensorboard',
        label: 'TensorBoard',
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

    const availableActions = getActionsForExperiment(experiment, headerActions, expPermissions);

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
        const cssProps: React.CSSProperties = { height: '32px', width: '32px' };
        switch (experiment.state) {
          case JobState.SCHEDULED:
          case JobState.SCHEDULEDBACKFILLED:
          case JobState.QUEUED:
          case RunState.Queued:
            cssProps['backgroundColor'] = 'white';
            cssProps['opacity'] = '0.25';
            break;
          case RunState.Running:
            cssProps['borderColor'] = 'white';
            break;
          default:
            break;
        }

        return isShownAnimation(experiment.state) ? (
          <>
            <ExperimentIcons isTooltipVisible={false} state={experiment.state} style={cssProps} />
            <div className={css.icon}>{iconNode}</div>
          </>
        ) : (
          <>{iconNode}</>
        );
      }
    },
    [experiment.state],
  );

  const foldableRows: InfoRow[] = useMemo(() => {
    const rows = [
      {
        content: (
          <Typography.Paragraph
            disabled={!experiment.description}
            ellipsis={{ rows: 1, tooltip: true }}
            style={{ margin: 0 }}>
            {experiment.description || 'N/A'}
          </Typography.Paragraph>
        ),
        label: 'Description',
      },
    ];
    if (experiment.forkedFrom && experiment.config.searcher.sourceTrialId) {
      rows.push({
        content: (
          <Link
            className={css.link}
            path={paths.trialDetails(experiment.config.searcher.sourceTrialId)}>
            Trial {experiment.config.searcher.sourceTrialId}
          </Link>
        ),
        label: 'Continued from',
      });
    }
    if (experiment.forkedFrom && !experiment.config.searcher.sourceTrialId) {
      rows.push({
        content: (
          <Link className={css.link} path={paths.experimentDetails(experiment.forkedFrom)}>
            Experiment {experiment.forkedFrom}
          </Link>
        ),
        label: 'Forked from',
      });
    }
    rows.push({ content: <TimeAgo datetime={experiment.startTime} long />, label: 'Started' });
    if (experiment.endTime != null) {
      rows.push({
        content: <TimeDuration duration={getDuration(experiment)} />,
        label: 'Duration',
      });
    }
    if (experiment.jobSummary && !terminalRunStates.has(experiment.state)) {
      rows.push({
        content: (
          <Link className={css.link} path={paths.jobs()}>
            {jobInfoLinkText}
          </Link>
        ),
        label: 'Job info',
      });
    }
    rows.push({
      content: (
        <div>
          {autoRestarts}
          {maxRestarts ? `/${maxRestarts}` : ''}
        </div>
      ),
      label: 'Auto restarts',
    });
    rows.push({
      content: (
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
      label: 'Tags',
    });

    return rows;
  }, [autoRestarts, disabled, experiment, experimentTags, jobInfoLinkText, maxRestarts]);

  return (
    <>
      <BreadcrumbBar experiment={experiment} id={experiment.id} type="experiment" />
      <PageHeaderFoldable
        foldableContent={<InfoBox rows={foldableRows} />}
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
                        icon={returnStatusIcon(<Icon name="pause" size="large" />)}
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
                        icon={returnStatusIcon(<Icon name="play" size="large" />)}
                        shape="circle"
                        onClick={handlePlayClick}
                      />
                    )}
                    {!isTerminated && (
                      <Button
                        className={css.buttonStop}
                        disabled={!canPausePlay}
                        icon={<Icon name="stop" size="large" />}
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
            {trial ? (
              <>
                <Icon name="arrow-right" size="tiny" />
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
        fetchExperimentDetails={fetchExperimentDetails}
      />
      {modalHyperparameterSearchContextHolder}
    </>
  );
};

export default ExperimentDetailsHeader;
