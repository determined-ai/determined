import { Button, Tooltip } from 'antd';
import Modal from 'antd/lib/modal/Modal';
import yaml from 'js-yaml';
import React, { useCallback, useState } from 'react';
import MonacoEditor from 'react-monaco-editor';
import TimeAgo from 'timeago-react';

import CheckpointModal from 'components/CheckpointModal';
import HumanReadableFloat from 'components/HumanReadableFloat';
import InfoBox, { InfoRow } from 'components/InfoBox';
import Link from 'components/Link';
import ProgressBar from 'components/ProgressBar';
import Section from 'components/Section';
import TagList from 'components/TagList';
import tagListCss from 'components/TagList.module.scss';
import useExperimentTags from 'hooks/useExperimentTags';
import { CheckpointDetail, ExperimentBase } from 'types';
import { getDuration, shortEnglishHumannizer } from 'utils/time';

import css from './ExperimentInfoBox.module.scss';

interface Props extends TopWorkloads {
  experiment: ExperimentBase;
  onTagsChange?: () => void;
}

export interface TopWorkloads {
  bestCheckpoint?: CheckpointDetail;
  bestValidation?: number;
}

const ExperimentInfoBox: React.FC<Props> = (
  { experiment, bestValidation, bestCheckpoint, onTagsChange }: Props,
) => {
  const config = experiment.config;
  const [ showConfig, setShowConfig ] = useState(false);
  const [ showBestCheckpoint, setShowBestCheckpoint ] = useState(false);

  const experimentTags = useExperimentTags(onTagsChange);
  const handleHideBestCheckpoint = useCallback(() => setShowBestCheckpoint(false), []);
  const handleHideConfig = useCallback(() => setShowConfig(false), []);
  const handleShowBestCheckpoint = useCallback(() => setShowBestCheckpoint(true), []);
  const handleShowConfig = useCallback(() => setShowConfig(true), []);

  const infoRows: InfoRow[] = [
    {
      content: !!experiment.progress && <ProgressBar
        percent={experiment.progress * 100}
        state={experiment.state} />,
      label: 'Progress',
    },
    {
      content: bestValidation !== undefined &&
        <>
          <HumanReadableFloat num={bestValidation} /> {`(${config.searcher.metric})`}
        </>,
      label: 'Best Validation',
    },
    {
      content:
          <Button onClick={handleShowConfig}>View Configuration</Button>,
      label: 'Configuration',
    },
    {
      content: bestCheckpoint && <Button onClick={handleShowBestCheckpoint}>
        Trial {bestCheckpoint.trialId} Batch {bestCheckpoint.batch}
      </Button>,
      label: 'Best Checkpoint',
    },
    {
      content: config.resources.maxSlots !== undefined ? config.resources.maxSlots : undefined,
      label: 'Max Slots',
    },
    {
      content: <Tooltip title={new Date(experiment.startTime).toLocaleString()}>
        <TimeAgo datetime={new Date(experiment.startTime)} />
      </Tooltip>,
      label: 'Start Time',
    },
    {
      content: experiment.endTime != null && shortEnglishHumannizer(getDuration(experiment)),
      label: 'Duration',
    },
    {
      content: <Link external isButton path={`/experiments/${experiment.id}/model_def`} rawLink>
        Download Model
      </Link>,
      label: 'Model Definition',
    },
    {
      content: <TagList
        className={tagListCss.noMargin}
        tags={experiment.config.labels || []}
        onChange={experimentTags.handleTagListChange(experiment.id)}
      />,
      label: 'Labels',
    },
  ];

  return (
    <Section bodyBorder maxHeight title="Summary">
      <InfoBox rows={infoRows} />
      {bestCheckpoint && <CheckpointModal
        checkpoint={bestCheckpoint}
        config={config}
        show={showBestCheckpoint}
        title={`Best Checkpoint for Experiment ${experiment.id}`}
        onHide={handleHideBestCheckpoint} />}
      <Modal
        bodyStyle={{ padding: 0 }}
        className={css.forkModal}
        footer={null}
        title={`Configuration for Experiment ${experiment.id}`}
        visible={showConfig}
        width={768}
        onCancel={handleHideConfig}>
        <MonacoEditor
          height="60vh"
          language="yaml"
          options={{
            minimap: { enabled: false },
            occurrencesHighlight: false,
            readOnly: true,
            scrollBeyondLastLine: false,
            selectOnLineNumbers: true,
          }}
          theme="vs-light"
          value={yaml.safeDump(experiment.configRaw)} />
      </Modal>
    </Section>
  );
};

export default ExperimentInfoBox;
