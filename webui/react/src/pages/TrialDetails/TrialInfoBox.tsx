import { Button, List, Modal } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import CheckpointModal from 'components/CheckpointModal';
import HumanReadableFloat from 'components/HumanReadableFloat';
import InfoBox from 'components/InfoBox';
import Section from 'components/Section';
import {
  Checkpoint, CheckpointDetail, CheckpointState, ExperimentDetails, HyperparameterValue, RunState,
  Step, TrialDetails, ValidationMetrics,
} from 'types';
import { isObject, numericSorter } from 'utils/data';
import { formatDatetime } from 'utils/date';
import { humanReadableBytes } from 'utils/string';
import { shortEnglishHumannizer } from 'utils/time';
import { checkpointSize, trialDurations } from 'utils/types';

import css from './TrialInfoBox.module.scss';

interface Props {
  trial: TrialDetails;
  experiment: ExperimentDetails;
}

const hyperparamsView = (params: Record<string, HyperparameterValue>) => {
  return <List
    dataSource={Object.entries(params)}
    renderItem={([ label, value ]) => {
      const textValue = isObject(value) ? JSON.stringify(value, null, 2) : value.toString();
      return <List.Item>{label}: {textValue}</List.Item>;
    }}
    size="small"
  />;
};

const TrialInfoBox: React.FC<Props> = ({ trial, experiment }: Props) => {
  const [ showHParams, setShowHParams ] = useState(false);
  const [ showBestCheckpoint, setShowBestCheckpoint ] = useState(false);

  const { metric, smallerIsBetter } = experiment.config.searcher || {};

  const bestValidation = useMemo(() => {
    const sortedValidations = trial.steps
      .filter(step => step.validation
        && step.validation.state === RunState.Completed
        && !!step.validation.metrics )
      .map(step => (step.validation?.metrics as ValidationMetrics)
        .validationMetrics[metric])
      .sort((a, b) => numericSorter(a, b, !smallerIsBetter));

    return sortedValidations[0];
  }, [ metric, smallerIsBetter, trial.steps ]);

  const bestCheckpoint: CheckpointDetail | undefined = useMemo(() => {
    const sortedSteps: Step[] = trial.steps
      .filter(step => step.checkpoint && step.checkpoint.state === CheckpointState.Completed)
      .sort((a, b) => numericSorter(
        a.checkpoint?.validationMetric,
        b.checkpoint?.validationMetric,
        !smallerIsBetter,
      ));
    const bestStep = sortedSteps[0];
    return bestStep ? {
      ...(bestStep.checkpoint as Checkpoint),
      batch: bestStep.numBatches + bestStep.priorBatchesProcessed,
      experimentId: experiment.id,
    } : undefined;
  }, [ experiment.id, smallerIsBetter, trial.steps ]);

  const totalCheckpointsSize = useMemo(() => {
    const totalBytes = trial.steps
      .filter(step => step.checkpoint
        && step.checkpoint.state === CheckpointState.Completed)
      .map(step =>checkpointSize(step.checkpoint as Checkpoint))
      .reduce((acc, cur) => acc + cur, 0);
    return humanReadableBytes(totalBytes);
  }, [ trial.steps ]);

  const durations = useMemo(() => trialDurations(trial.steps), [ trial.steps ]);

  const handleShowBestCheckpoint = useCallback(() => setShowBestCheckpoint(true), []);
  const handleHideBestCheckpoint = useCallback(() => setShowBestCheckpoint(false), []);
  const handleShowHParams = useCallback(() => setShowHParams(true), []);
  const handleHideHParams = useCallback(() => setShowHParams(false), []);

  const infoRows = [
    {
      content: formatDatetime(trial.startTime),
      label: 'Start Time',
    },
    {
      content: trial.endTime && formatDatetime(trial.endTime),
      label: 'End Time',
    },
    {
      content: <div className={css.duration}>
        <div>Training: {shortEnglishHumannizer(durations.train)}</div>
        <div>Checkpointing: {shortEnglishHumannizer(durations.checkpoint)}</div>
        <div>Validating: {shortEnglishHumannizer(durations.validation)}</div>
      </div>,
      label: 'Durations',
    },
    {
      content: bestValidation &&
        <>
          <HumanReadableFloat num={bestValidation} /> {`(${experiment.config.searcher.metric})`}
        </>,
      label: 'Best Validation',
    },
    {
      content: bestCheckpoint && (<>
        <Button onClick={handleShowBestCheckpoint}>
          Trial {bestCheckpoint.trialId} Batch {bestCheckpoint.batch}
        </Button>
        <CheckpointModal
          checkpoint={bestCheckpoint}
          config={experiment.config}
          show={showBestCheckpoint}
          title={`Best Checkpoint for Trial ${trial.id}`}
          onHide={handleHideBestCheckpoint} />
      </>),
      label: 'Best Checkpoint',
    },
    {
      content: totalCheckpointsSize,
      label: 'Total Checkpoint Size',
    },
    {
      content: <Button onClick={handleShowHParams}>Show</Button>,
      label: 'H-params',
    },
  ];

  return (
    <Section bodyBorder maxHeight title="Summary">
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
