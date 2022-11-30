import { Button, Space } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import BreadcrumbBar from 'components/BreadcrumbBar';
import ExperimentIcons from 'components/ExperimentIcons';
import InlineEditor from 'components/InlineEditor';
import Link from 'components/Link';
import PageHeaderFoldable, { Option } from 'components/PageHeaderFoldable';
import TagList from 'components/TagList';
import TimeAgo from 'components/TimeAgo';
import TimeDuration from 'components/TimeDuration';
import { pausableRunStates, stateToLabel, terminalRunStates } from 'constants/states';
import useExperimentTags from 'hooks/useExperimentTags';
import useModalExperimentCreate, {
  CreateExperimentType,
} from 'hooks/useModal/Experiment/useModalExperimentCreate';
import useModalExperimentDelete from 'hooks/useModal/Experiment/useModalExperimentDelete';
import useModalExperimentMove from 'hooks/useModal/Experiment/useModalExperimentMove';
import useModalExperimentStop from 'hooks/useModal/Experiment/useModalExperimentStop';
import useModalHyperparameterSearch from 'hooks/useModal/HyperparameterSearch/useModalHyperparameterSearch';
import usePermissions from 'hooks/usePermissions';
import ExperimentHeaderProgress from 'pages/ExperimentDetails/Header/ExperimentHeaderProgress';
import { handlePath, paths } from 'routes/utils';
import {
  activateExperiment,
  archiveExperiment,
  openOrCreateTensorBoard,
  patchExperiment,
  pauseExperiment,
  unarchiveExperiment,
} from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import Spinner from 'shared/components/Spinner/Spinner';
import { getDuration } from 'shared/utils/datetime';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { getStateColorCssVar } from 'themes';
import { ExperimentAction as Action, ExperimentBase, RunState, TrialItem } from 'types';
import handleError from 'utils/error';
import { canActionExperiment, getActionsForExperiment } from 'utils/experiment';
import { openCommandResponse } from 'utils/wait';

import css from './ExperimentDetailsHeader.module.scss';

interface Props {
  experiment: ExperimentBase;
  fetchExperimentDetails: () => void;
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

  const handleModalClose = useCallback(() => fetchExperimentDetails(), [fetchExperimentDetails]);

  const expPermissions = usePermissions();
  const isMovable =
    canActionExperiment(Action.Move, experiment) &&
    expPermissions.canMoveExperiment({ experiment });
  const canPausePlay = expPermissions.canModifyExperiment({
    workspace: { id: experiment.workspaceId },
  });

  const { contextHolder: modalExperimentStopContextHolder, modalOpen: openModalStop } =
    useModalExperimentStop({ experimentId: experiment.id, onClose: handleModalClose });

  const { contextHolder: modalExperimentMoveContextHolder, modalOpen: openModalMove } =
    useModalExperimentMove({ onClose: handleModalClose });

  const { contextHolder: modalExperimentDeleteContextHolder, modalOpen: openModalDelete } =
    useModalExperimentDelete({ experiment: experiment });

  const { contextHolder: modalExperimentCreateContextHolder, modalOpen: openModalCreate } =
    useModalExperimentCreate();

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

  const handleStopClick = useCallback(() => openModalStop(), [openModalStop]);

  const handleDeleteClick = useCallback(() => openModalDelete(), [openModalDelete]);

  const handleMoveClick = useCallback(
    () =>
      openModalMove({
        experimentIds: isMovable ? [experiment.id] : undefined,
        sourceProjectId: experiment.projectId,
        sourceWorkspaceId: experiment.workspaceId,
      }),
    [openModalMove, experiment, isMovable],
  );

  const handleContinueTrialClick = useCallback(() => {
    openModalCreate({
      experiment,
      trial,
      type: CreateExperimentType.ContinueTrial,
    });
  }, [experiment, openModalCreate, trial]);

  const handleForkClick = useCallback(() => {
    openModalCreate({ experiment, type: CreateExperimentType.Fork });
  }, [experiment, openModalCreate]);

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

  const handleDescriptionUpdate = useCallback(
    async (newValue: string) => {
      try {
        await patchExperiment({ body: { description: newValue }, experimentId: experiment.id });
        await fetchExperimentDetails();
      } catch (e) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to update experiment description.',
          silent: false,
          type: ErrorType.Server,
        });
        return e as Error;
      }
    },
    [experiment.id, fetchExperimentDetails],
  );

  const handleNameUpdate = useCallback(
    async (newValue: string) => {
      try {
        await patchExperiment({ body: { name: newValue }, experimentId: experiment.id });
        await fetchExperimentDetails();
      } catch (e) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to update experiment name.',
          silent: false,
          type: ErrorType.Server,
        });
        return e as Error;
      }
    },
    [experiment.id, fetchExperimentDetails],
  );

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
        onClick: handleContinueTrialClick,
      },
      [Action.Delete]: {
        isLoading: isRunningDelete,
        key: 'delete',
        label: 'Delete',
        onClick: handleDeleteClick,
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
        onClick: handleForkClick,
      },
      [Action.Move]: {
        key: 'move',
        label: 'Move',
        onClick: handleMoveClick,
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
    handleContinueTrialClick,
    isRunningDelete,
    handleDeleteClick,
    handleHyperparameterSearch,
    handleForkClick,
    handleMoveClick,
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

  return (
    <>
      <BreadcrumbBar experiment={experiment} id={experiment.id} type="experiment" />
      <PageHeaderFoldable
        foldableContent={
          <div className={css.foldableSection}>
            <div className={css.foldableItem}>
              <span className={css.foldableItemLabel}>Description:</span>
              <InlineEditor
                allowNewline
                disabled={disabled}
                maxLength={500}
                placeholder={disabled ? 'Archived' : 'Add description...'}
                style={{ minWidth: 120 }}
                value={experiment.description || ''}
                onSave={handleDescriptionUpdate}
              />
            </div>
            <div className={css.foldableItem}>
              <span className={css.foldableItemLabel}>Start Time:</span>
              <TimeAgo datetime={experiment.startTime} long />
            </div>
            {experiment.endTime != null && (
              <div className={css.foldableItem}>
                <span className={css.foldableItemLabel}>Duration:</span>
                <TimeDuration duration={getDuration(experiment)} />
              </div>
            )}
            {experiment.jobSummary && !terminalRunStates.has(experiment.state) && (
              <div className={css.foldableItem}>
                <span className={css.foldableItemLabel}>Job Info:</span>
                <Link className={css.link} path={paths.jobs()}>
                  {jobInfoLinkText}
                </Link>
              </div>
            )}
            <div className={css.foldableItem}>
              <span className={css.foldableItemLabel}>Auto Restarts:</span>
              <span>
                {autoRestarts}
                {maxRestarts ? `/${maxRestarts}` : ''}
              </span>
            </div>
            <TagList
              disabled={disabled}
              ghost={true}
              tags={experiment.config.labels || []}
              onChange={experimentTags.handleTagListChange(experiment.id)}
            />
          </div>
        }
        leftContent={
          <Space align="center" className={css.base}>
            <Spinner spinning={isChangingState}>
              <div className={css.stateIcon}>
                {<ExperimentIcons state={experiment.state} />}
                <div className={classes.join(' ')} style={stateStyle}>
                  {isPausable && (
                    <Button
                      className={css.buttonPause}
                      disabled={!canPausePlay}
                      icon={<Icon name="pause" size="large" />}
                      shape="circle"
                      onClick={handlePauseClick}
                    />
                  )}
                  {isPaused && (
                    <Button
                      className={css.buttonPlay}
                      disabled={!canPausePlay}
                      icon={<Icon name="play" size="large" />}
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
                      onClick={handleStopClick}
                    />
                  )}
                  <label>{stateToLabel(experiment.state)}</label>
                </div>
              </div>
            </Spinner>
            <div className={css.id}>Experiment {experiment.id}</div>
            <div className={css.name}>
              <InlineEditor
                disabled={disabled}
                maxLength={128}
                placeholder="experiment name"
                value={experiment.name}
                onSave={handleNameUpdate}
              />
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
      {modalExperimentCreateContextHolder}
      {modalExperimentDeleteContextHolder}
      {modalExperimentMoveContextHolder}
      {modalExperimentStopContextHolder}
      {modalHyperparameterSearchContextHolder}
    </>
  );
};

export default ExperimentDetailsHeader;
