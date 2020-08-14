import { Button, Tooltip } from 'antd';
import Modal from 'antd/lib/modal/Modal';
import yaml from 'js-yaml';
import React, { useCallback, useMemo, useState } from 'react';
import MonacoEditor from 'react-monaco-editor';
import TimeAgo from 'timeago-react';

import CheckpointModal from 'components/CheckpointModal';
import InfoBox, { InfoRow } from 'components/InfoBox';
import Link from 'components/Link';
import ProgressBar from 'components/ProgressBar';
import Section from 'components/Section';
import { CheckpointDetail, CheckpointState, ExperimentDetails } from 'types';
import { humanReadableFloat } from 'utils/string';
import { getDuration, shortEnglishHumannizer } from 'utils/time';

import css from './ExperimentInfoBox.module.scss';

interface Props {
  experiment: ExperimentDetails;
}

const ExperimentInfoBox: React.FC<Props> = ({ experiment }: Props) => {
  const config = experiment.config;
  const [ showConfig, setShowConfig ] = useState(false);
  const [ showBestCheckpoint, setShowBestCheckpoint ] = useState(false);

  const orderFactor = experiment.config.searcher.smallerIsBetter ? 1 : -1;

  const bestValidation = useMemo(() => {
    const sortedValidations = experiment.validationHistory
      .filter(a => a.validationError !== undefined)
      .sort((a, b) => (a.validationError as number - (b.validationError as number)) * orderFactor);
    return sortedValidations[0]?.validationError;
  }, [ experiment.validationHistory, orderFactor ]);

  const bestCheckpoint: CheckpointDetail | undefined = useMemo(() => {
    const sortedCheckpoints: CheckpointDetail[] = experiment.trials
      .filter(trial => trial.bestAvailableCheckpoint
        && trial.bestAvailableCheckpoint.validationMetric
        && trial.bestAvailableCheckpoint.state === CheckpointState.Completed)
      .map(trial => ({
        ...trial.bestAvailableCheckpoint,
        batch: trial.totalBatchesProcessed,
        experimentId: trial.experimentId,
        trialId: trial.id,
      }) as CheckpointDetail)
      .sort((a, b) => {
        return (a.validationMetric as number - (b.validationMetric as number)) * orderFactor;
      });
    return sortedCheckpoints[0];
  }, [ experiment.trials, orderFactor ]);

  const handleShowBestCheckpoint = useCallback(() => setShowBestCheckpoint(true), []);
  const handleHideBestCheckpoint = useCallback(() => setShowBestCheckpoint(false), []);
  const handleShowConfig = useCallback(() => setShowConfig(true), []);
  const handleHideConfig = useCallback(() => setShowConfig(false), []);

  const infoRows: InfoRow[] = [
    {
      content: experiment.progress && <ProgressBar
        percent={experiment.progress * 100}
        state={experiment.state} />,
      label: 'Progress',
    },
    {
      content: bestValidation &&
      `${humanReadableFloat(bestValidation)} (${config.searcher.metric})`,
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
      content: <Link isButton path={`/experiments/${experiment.id}/model_def`}>
        Download Model
      </Link>,
      label: 'Model Definition',
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
