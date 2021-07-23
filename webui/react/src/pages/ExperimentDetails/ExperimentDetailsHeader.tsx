import { Tooltip } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import TimeAgo from 'timeago-react';

import Icon from 'components/Icon';
import InlineTextEdit from 'components/InlineTextEdit';
import PageHeaderFoldable, { Option } from 'components/PageHeaderFoldable';
import TagList from 'components/TagList';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import useExperimentTags from 'hooks/useExperimentTags';
import ExperimentHeaderProgress from 'pages/ExperimentDetails/Header/ExperimentHeaderProgress';
import ExperimentState from 'pages/ExperimentDetails/Header/ExperimentHeaderState';
import { handlePath, paths } from 'routes/utils';
import {
  archiveExperiment, openOrCreateTensorboard, patchExperiment, unarchiveExperiment,
} from 'services/api';
import { getStateColorCssVar } from 'themes';
import { ExperimentBase } from 'types';
import { getDuration, shortEnglishHumannizer } from 'utils/time';
import { terminalRunStates } from 'utils/types';
import { openCommand } from 'wait';

import css from './ExperimentDetailsHeader.module.scss';

interface Props {
  experiment: ExperimentBase;
  fetchExperimentDetails: () => void;
  isSingleTrial: boolean;
  showContinueTrial: () => void;
  showForkModal: () => void;
}

const ExperimentDetailsHeader: React.FC<Props> = ({
  showContinueTrial,
  experiment,
  fetchExperimentDetails,
  isSingleTrial,
  showForkModal,
}: Props) => {
  const [ isRunningArchive, setIsRunningArchive ] = useState<boolean>(false);
  const [ isRunningTensorboard, setIsRunningTensorboard ] = useState<boolean>(false);
  const [ isRunningUnarchive, setIsRunningUnarchive ] = useState<boolean>(false);
  const experimentTags = useExperimentTags(fetchExperimentDetails);

  useEffect(() => {
    setIsRunningArchive(false);
    setIsRunningUnarchive(false);
  }, [ experiment.archived ]);

  const handleDescriptionUpdate = useCallback(async (newValue: string) => {
    try {
      await patchExperiment({ body: { description: newValue }, experimentId: experiment.id });
      await fetchExperimentDetails();
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to update experiment description.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ experiment.id, fetchExperimentDetails ]);

  const headerOptions = useMemo<Option[]>(() => {
    const continueTrial: Option = {
      key: 'continue-trial',
      label: 'Continue Trial',
      onClick: showContinueTrial,
    };
    const downloadModel: Option = {
      icon: <Icon name="download" size="small" />,
      key: 'download-model',
      label: 'Download Model',
      onClick: (e) => {
        handlePath(e, { external: true, path: paths.experimentModelDef(experiment.id) });
      },
    };
    const fork: Option = {
      icon: <Icon name="fork" size="small" />,
      key: 'fork',
      label: 'Fork',
      onClick: showForkModal,
    };
    const tensorboard: Option = {
      icon: <Icon name="tensorboard" size="small" />,
      isLoading: isRunningTensorboard,
      key: 'tensorboard',
      label: 'TensorBoard',
      onClick: async () => {
        setIsRunningTensorboard(true);
        try {
          const tensorboard = await openOrCreateTensorboard({ experimentIds: [ experiment.id ] });
          openCommand(tensorboard);
          setIsRunningTensorboard(false);
        } catch (e) {
          setIsRunningTensorboard(false);
        }
      },
    };

    const options: Option[] = isSingleTrial ? [
      tensorboard,
      downloadModel,
      fork,
      continueTrial,
    ] : [
      fork,
      tensorboard,
      downloadModel,
    ];

    if (terminalRunStates.has(experiment.state)) {
      if (experiment.archived) {
        options.push({
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
        });
      } else {
        options.push({
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
        });
      }
    }

    return options;
  }, [
    experiment.archived,
    experiment.id,
    experiment.state,
    fetchExperimentDetails,
    isRunningArchive,
    isRunningTensorboard,
    isRunningUnarchive,
    isSingleTrial,
    showContinueTrial,
    showForkModal,
  ]);

  return (
    <>
      <PageHeaderFoldable
        foldableContent={<>
          <div className={css.foldableItem}>
            <span className={css.foldableItemLabel}>Description:</span>
            <InlineTextEdit
              setValue={handleDescriptionUpdate}
              value={experiment.description || ''}
            />
          </div>
          <div className={css.foldableItem}>
            <span className={css.foldableItemLabel}>Start Time:</span>
            <Tooltip title={new Date(experiment.startTime).toLocaleString()}>
              <TimeAgo datetime={new Date(experiment.startTime)} />
            </Tooltip>
          </div>
          {experiment.endTime != null && (
            <div className={css.foldableItem}>
              <span className={css.foldableItemLabel}>Duration:</span>
              {shortEnglishHumannizer(getDuration(experiment))}
            </div>
          )}
          <TagList
            ghost={true}
            tags={experiment.config.labels || []}
            onChange={experimentTags.handleTagListChange(experiment.id)}
          />
        </>}
        leftContent={<>
          <ExperimentState experiment={experiment} />
          <div className={css.experimentTitle}>
            Experiment {experiment.id} | <span>{experiment.name}</span>
          </div>
        </>}
        options={headerOptions}
        style={{ backgroundColor: getStateColorCssVar(experiment.state) }}
      />
      <ExperimentHeaderProgress experiment={experiment} />
    </>
  );
};

export default ExperimentDetailsHeader;
