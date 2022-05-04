import { Button } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Icon from 'components/Icon';
import InlineEditor from 'components/InlineEditor';
import Link from 'components/Link';
import PageHeaderFoldable, { Option } from 'components/PageHeaderFoldable';
import Spinner from 'components/Spinner';
import TagList from 'components/TagList';
import TimeAgo from 'components/TimeAgo';
import TimeDuration from 'components/TimeDuration';
import {
  deletableRunStates,
  pausableRunStates,
  stateToLabel,
  terminalRunStates,
} from 'constants/states';
import useExperimentTags from 'hooks/useExperimentTags';
import useModalExperimentCreate, {
  CreateExperimentType,
} from 'hooks/useModal/useModalExperimentCreate';
import useModalExperimentDelete from 'hooks/useModal/useModalExperimentDelete';
import useModalExperimentStop from 'hooks/useModal/useModalExperimentStop';
import ExperimentHeaderProgress from 'pages/ExperimentDetails/Header/ExperimentHeaderProgress';
import { handlePath, paths } from 'routes/utils';
import {
  activateExperiment,
  archiveExperiment, openOrCreateTensorBoard, patchExperiment,
  pauseExperiment,
  unarchiveExperiment,
} from 'services/api';
import { getStateColorCssVar } from 'themes';
import { DetailedUser, ExperimentBase, RecordKey, RunState, TrialDetails } from 'types';
import { getDuration } from 'utils/datetime';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { openCommand } from 'wait';

import css from './ExperimentDetailsHeader.module.scss';

interface Props {
  curUser?: DetailedUser;
  experiment: ExperimentBase;
  fetchExperimentDetails: () => void;
  trial?: TrialDetails;
}

const ExperimentDetailsHeader: React.FC<Props> = ({
  curUser,
  experiment,
  fetchExperimentDetails,
  trial,
}: Props) => {
  const [ isChangingState, setIsChangingState ] = useState(false);
  const [ isRunningArchive, setIsRunningArchive ] = useState<boolean>(false);
  const [ isRunningTensorBoard, setIsRunningTensorBoard ] = useState<boolean>(false);
  const [ isRunningUnarchive, setIsRunningUnarchive ] = useState<boolean>(false);
  const [ isRunningDelete, setIsRunningDelete ] = useState<boolean>(
    experiment.state === RunState.Deleting,
  );
  const experimentTags = useExperimentTags(fetchExperimentDetails);

  const handleModalClose = useCallback(() => fetchExperimentDetails(), [ fetchExperimentDetails ]);

  const { modalOpen: openModalStop } = useModalExperimentStop({
    experimentId: experiment.id,
    onClose: handleModalClose,
  });

  const { modalOpen: openModalDelete } = useModalExperimentDelete({ experimentId: experiment.id });

  const { modalOpen: openModalCreate } = useModalExperimentCreate();

  const backgroundColor = useMemo(() => {
    return getStateColorCssVar(experiment.state);
  }, [ experiment.state ]);

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
  }, [ experiment.id, fetchExperimentDetails ]);

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
  }, [ experiment.id, fetchExperimentDetails ]);

  const handleStopClick = useCallback(() => openModalStop(), [ openModalStop ]);

  const handleDeleteClick = useCallback(() => openModalDelete(), [ openModalDelete ]);

  const handleContinueTrialClick = useCallback(() => {
    openModalCreate({ experiment, trial, type: CreateExperimentType.ContinueTrial });
  }, [ experiment, openModalCreate, trial ]);

  const handleForkClick = useCallback(() => {
    openModalCreate({ experiment, type: CreateExperimentType.Fork });
  }, [ experiment, openModalCreate ]);

  useEffect(() => {
    setIsRunningArchive(false);
    setIsRunningUnarchive(false);
  }, [ experiment.archived ]);

  useEffect(() => {
    setIsRunningDelete(experiment.state === RunState.Deleting);
  }, [ experiment.state ]);

  const handleDescriptionUpdate = useCallback(async (newValue: string) => {
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
  }, [ experiment.id, fetchExperimentDetails ]);

  const handleNameUpdate = useCallback(async (newValue: string) => {
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
  }, [ experiment.id, fetchExperimentDetails ]);

  const headerOptions = useMemo(() => {
    const options: Record<RecordKey, Option> = {
      archive: {
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
      continueTrial: {
        key: 'continue-trial',
        label: 'Continue Trial',
        onClick: handleContinueTrialClick,
      },
      delete: {
        icon: <Icon name="fork" size="small" />,
        isLoading: isRunningDelete,
        key: 'delete',
        label: 'Delete',
        onClick: handleDeleteClick,
      },
      downloadModel: {
        icon: <Icon name="download" size="small" />,
        key: 'download-model',
        label: 'Download Experiment Code',
        onClick: (e) => {
          handlePath(e, { external: true, path: paths.experimentModelDef(experiment.id) });
        },
      },
      fork: {
        icon: <Icon name="fork" size="small" />,
        key: 'fork',
        label: 'Fork',
        onClick: handleForkClick,
      },
      tensorboard: {
        icon: <Icon name="tensor-board" size="small" />,
        isLoading: isRunningTensorBoard,
        key: 'tensorboard',
        label: 'TensorBoard',
        onClick: async () => {
          setIsRunningTensorBoard(true);
          try {
            const tensorboard = await openOrCreateTensorBoard({ experimentIds: [ experiment.id ] });
            openCommand(tensorboard);
            setIsRunningTensorBoard(false);
          } catch (e) {
            setIsRunningTensorBoard(false);
          }
        },
      },
      unarchive: {
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
    return [
      options.fork,
      trial?.id && options.continueTrial,
      options.tensorboard,
      options.downloadModel,
      terminalRunStates.has(experiment.state) && (
        experiment.archived ? options.archive : options.unarchive
      ),
      deletableRunStates.has(experiment.state) &&
        curUser && (curUser.isAdmin || curUser.id === experiment.userId) && options.delete,
    ].filter(option => !!option) as Option[];
  }, [
    curUser,
    isRunningDelete,
    experiment.archived,
    experiment.id,
    experiment.state,
    experiment.userId,
    fetchExperimentDetails,
    handleContinueTrialClick,
    handleDeleteClick,
    handleForkClick,
    isRunningArchive,
    isRunningTensorBoard,
    isRunningUnarchive,
    trial?.id,
  ]);

  const jobInfoLinkText = useMemo(() => {
    if (!experiment.jobSummary) return 'Not available';
    const isJobOrderAvailable = experiment.jobSummary.jobsAhead >= 0;
    const isFirstJob = experiment.jobSummary.jobsAhead === 0;
    if (!isJobOrderAvailable) return 'Available here';
    if (isFirstJob) return 'No jobs ahead of this one';
    return `${experiment.jobSummary.jobsAhead} jobs ahead of this one`;
  }, [ experiment.jobSummary ]);

  return (
    <>
      <PageHeaderFoldable
        foldableContent={(
          <div className={css.foldableSection}>
            <div className={css.foldableItem}>
              <span className={css.foldableItemLabel}>Description:</span>
              <InlineEditor
                allowNewline
                isOnDark
                maxLength={500}
                placeholder="Add description"
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
                <Link className={css.link} path={paths.jobs()}>{jobInfoLinkText}</Link>
              </div>
            )}
            <TagList
              ghost={true}
              tags={experiment.config.labels || []}
              onChange={experimentTags.handleTagListChange(experiment.id)}
            />
          </div>
        )}
        leftContent={(
          <div className={css.base}>
            <div className={css.experimentInfo}>
              <Spinner spinning={isChangingState}>
                <div className={css.experimentState} style={{ backgroundColor }}>
                  {pausableRunStates.has(experiment.state) && (
                    <Button
                      className={css.buttonPause}
                      ghost={true}
                      icon={<Icon name="pause" size="large" />}
                      shape="circle"
                      onClick={handlePauseClick}
                    />
                  )}
                  {experiment.state === RunState.Paused && (
                    <Button
                      className={css.buttonPlay}
                      ghost={true}
                      icon={<Icon name="play" size="large" />}
                      shape="circle"
                      onClick={handlePlayClick}
                    />
                  )}
                  {!terminalRunStates.has(experiment.state) && (
                    <Button
                      className={css.buttonStop}
                      ghost={true}
                      icon={<Icon name="stop" size="large" />}
                      shape="circle"
                      onClick={handleStopClick}
                    />
                  )}
                  <span className={css.state}>{stateToLabel(experiment.state)}</span>
                </div>
              </Spinner>
              <div className={css.experimentId}>Experiment {experiment.id}</div>
            </div>
            <div className={css.experimentName}>
              <InlineEditor
                isOnDark
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
          </div>
        )}
        options={headerOptions}
        style={{ backgroundColor: getStateColorCssVar(experiment.state) }}
      />
      <ExperimentHeaderProgress experiment={experiment} />
    </>
  );
};

export default ExperimentDetailsHeader;
