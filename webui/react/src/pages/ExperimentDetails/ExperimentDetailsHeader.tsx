import React, { useEffect, useMemo, useState } from 'react';

import Icon from 'components/Icon';
import PageHeaderFoldable, { Option } from 'components/PageHeaderFoldable';
import TagList from 'components/TagList';
import useExperimentTags from 'hooks/useExperimentTags';
import ExperimentHeaderProgress from 'pages/ExperimentDetails/Header/ExperimentHeaderProgress';
import ExperimentState from 'pages/ExperimentDetails/Header/ExperimentHeaderState';
import { handlePath, paths } from 'routes/utils';
import { archiveExperiment, openOrCreateTensorboard, unarchiveExperiment } from 'services/api';
import { getStateColorCssVar } from 'themes';
import { ExperimentBase } from 'types';
import { terminalRunStates } from 'utils/types';
import { openCommand } from 'wait';

import css from './ExperimentDetailsHeader.module.scss';

interface Props {
  experiment: ExperimentBase;
  fetchExperimentDetails: () => void;
  showForkModal: () => void;
}

const ExperimentDetailsHeader: React.FC<Props> = (
  { experiment, fetchExperimentDetails, showForkModal }: Props,
) => {
  const [ isRunningArchive, setIsRunningArchive ] = useState<boolean>(false);
  const [ isRunningTensorboard, setIsRunningTensorboard ] = useState<boolean>(false);
  const [ isRunningUnarchive, setIsRunningUnarchive ] = useState<boolean>(false);
  const experimentTags = useExperimentTags(fetchExperimentDetails);

  useEffect(() => {
    setIsRunningArchive(false);
    setIsRunningUnarchive(false);
  }, [ experiment.archived ]);

  const headerOptions = useMemo<Option[]>(() => {
    const options: Option[] = [
      {
        icon: <Icon name="fork" size="small" />,
        key: 'fork',
        label: 'Fork',
        onClick: showForkModal,
      },
      {
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
      },
      {
        key: 'download-model',
        label: 'Download Model',
        onClick: (e) => {
          handlePath(e, { external: true, path: paths.experimentModelDef(experiment.id) });
        },
      },
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
    showForkModal,
  ]);

  return (
    <>
      <PageHeaderFoldable
        foldableContent={<>
          <div className={css.experimentName}>
            <span>Name:</span> {experiment.name}
          </div>
          <div className={css.experimentDescription}>
            <span>Description:</span> {experiment.description}
          </div>
          <TagList
            ghost={true}
            tags={experiment.config.labels || []}
            onChange={experimentTags.handleTagListChange(experiment.id)}
          />
        </>}
        leftContent={<>
          <ExperimentState experiment={experiment} />
          <div className={css.experimentTitle}>
            Experiment {experiment.id} <span>({experiment.name})</span>
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
