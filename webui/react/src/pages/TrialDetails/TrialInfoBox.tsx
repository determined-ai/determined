import { Button, List, Modal } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import CheckpointModal from 'components/CheckpointModal';
import InfoBox from 'components/InfoBox';
import Section from 'components/Section';
import { Checkpoint, CheckpointState, ExperimentDetails, RunState, TrialDetails,
  ValidationMetrics } from 'types';
import { formatDatetime } from 'utils/date';
import { humanReadableBytes, humanReadableFloat } from 'utils/string';
import { shortEnglishHumannizer } from 'utils/time';
import { checkpointSize, trialDurations } from 'utils/types';

import css from './TrialInfoBox.module.scss';

interface Props {
  trial: TrialDetails;
  experiment: ExperimentDetails;
}

const hyperparamsView = (params: Record<string, React.ReactNode>) => {
  return <List
    dataSource={Object.entries(params)}
    renderItem={([ label, value ]) => <List.Item>{label}: {value}</List.Item>}
    size="small"
  />;
};

const TrialInfoBox: React.FC<Props> = ({ trial, experiment }: Props) => {
  const [ showHParams, setShowHParams ] = useState(false);
  const [ showBestCheckpoint, setShowBestCheckpoint ] = useState(false);

  const orderFactor = experiment.config.searcher.smallerIsBetter ? 1 : -1;

  const bestValidation = useMemo(() => {
    const sortedValidations = trial.steps
      .filter(step => step.validation
        && step.validation.state === RunState.Completed
        && !!step.validation.metrics )
      .map(step => (step.validation?.metrics as ValidationMetrics)
        .validationMetrics[experiment.config.searcher.metric])
      .sort((a, b) => (a - b) * orderFactor);

    return sortedValidations[0];
  }, [ trial.steps, orderFactor, experiment.config.searcher.metric ]);

  const bestCheckpoint: Checkpoint = useMemo(() => {
    const sortedCheckpoints: Checkpoint[] = trial.steps
      .filter(step => step.checkpoint && step.checkpoint.state === CheckpointState.Completed)
      .map(step => step.checkpoint as Checkpoint)
      .sort((a, b) => {
        return (a.validationMetric as number - (b.validationMetric as number)) * orderFactor;
      });
    return sortedCheckpoints[0];
  }, [ trial.steps, orderFactor ]);

  const handleShowBestCheckpoint = useCallback(() => setShowBestCheckpoint(true), []);
  const handleHideBestCheckpoint = useCallback(() => setShowBestCheckpoint(false), []);
  const handleShowHParams = useCallback(() => setShowHParams(true), []);
  const handleHideHParams = useCallback(() => setShowHParams(false), []);

  const totalCheckpointsSize = useMemo(() => {
    const totalBytes = trial.steps
      .filter(step => step.checkpoint
        && step.checkpoint.state === CheckpointState.Completed)
      .map(step =>checkpointSize(step.checkpoint as Checkpoint))
      .reduce((acc, cur) => acc + cur, 0);
    return humanReadableBytes(totalBytes);
  }, [ trial.steps ]);

  const durations = trialDurations(trial.steps);

  const infoRows = [
    {
      content: <Badge state={experiment.state} type={BadgeType.State} />,
      label: 'State',
    },
    {
      content: formatDatetime(experiment.startTime),
      label: 'Start Time',
    },
    {
      content: trial.endTime && formatDatetime(experiment.startTime),
      label: 'End Time',
    },
    {
      content: <ul className={css.duration}>
        <li>Training: {shortEnglishHumannizer(durations.train)}</li>
        <li>Checkpointing: {shortEnglishHumannizer(durations.checkpoint)}</li>
        <li>Validation: {shortEnglishHumannizer(durations.validation)}</li>
      </ul>,
      label: 'Durations',
    },
    {
      content: bestValidation &&
        `${humanReadableFloat(bestValidation)} (${experiment.config.searcher.metric})`,
      label: 'Best Validation',
    },
    {
      content: <Button onClick={handleShowHParams}>Show</Button>,
      label: 'H-params',
    },
    {
      content: bestCheckpoint && (<>
        <Button onClick={handleShowBestCheckpoint}>
          {/* FIXME remove steps project */}
              Trial {bestCheckpoint.trialId} Step ID {bestCheckpoint.stepId}
        </Button>
        <CheckpointModal
          // FIXME remove steps project
          checkpoint={{ ...bestCheckpoint, batch: bestCheckpoint.stepId }}
          config={experiment.config}
          show={showBestCheckpoint}
          title={`Best Checkpoint for Trial ${trial.id}`}
          onHide={handleHideBestCheckpoint} />
      </>),
      label: 'Best Checkpoint',
    },
    {
      content: totalCheckpointsSize,
      label: 'Checkpoint Size',
    },
  ];

  return (
    <Section bodyBorder maxHeight title="Info Box">
      <InfoBox rows={infoRows} />
      <Modal
        bodyStyle={{ padding: 0 }}
        cancelButtonProps={{ style: { display: 'none' } }}
        title={`Trial ${trial.id} Hyperparameters`}
        visible={showHParams}
        onCancel={handleHideHParams}
        onOk={handleHideHParams}>
        {hyperparamsView(trial.hparams)}
      </Modal>
    </Section>
  );
};

export default TrialInfoBox;
