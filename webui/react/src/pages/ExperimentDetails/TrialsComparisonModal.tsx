import { Button, Tag, Tooltip } from 'antd';
import Modal from 'antd/lib/modal/Modal';
import axios from 'axios';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import BadgeTag from 'components/BadgeTag';
import CheckpointModal from 'components/CheckpointModal';
import HumanReadableFloat from 'components/HumanReadableFloat';
import Icon from 'components/Icon';
import { getTrialDetails } from 'services/api';
import { ApiState } from 'services/types';
import { isAborted } from 'services/utils';
import { CheckpointState, CheckpointWorkload,
  CheckpointWorkloadExtended,
  ExperimentBase,
  MetricName,
  MetricType, TrialDetails, TrialItem } from 'types';
import { humanReadableBytes } from 'utils/string';
import { getDuration, shortEnglishHumannizer } from 'utils/time';
import { extractMetricNames, trialDurations, TrialDurations } from 'utils/trial';
import { checkpointSize } from 'utils/types';

import css from './TrialsComparisonModal.module.scss';

interface ModalProps {
  experiment: ExperimentBase;
  onCancel: () => void;
  onUnselect: (trialId: number) => void;
  trials: TrialItem[];
  visible: boolean;
}

interface TableProps {
  experiment: ExperimentBase;
  onUnselect: (trialId: number) => void;
  trials: TrialItem[];
}

const TrialsComparisonModal: React.FC<ModalProps> =
({ experiment, onCancel, onUnselect, trials, visible }: ModalProps) => {
  return (
    <Modal
      footer={null}
      title={`Experiment ${trials.first()?.experimentId} Trial Comparison`}
      visible={visible}
      width={1200}
      onCancel={onCancel}>
      <TrialsComparisonTable experiment={experiment} trials={trials} onUnselect={onUnselect} />
    </Modal>
  );
};

const TrialsComparisonTable: React.FC<TableProps> = (
  { trials, experiment, onUnselect }: TableProps,
) => {
  const [ trialsDetails, setTrialsDetails ] = useState<Record<string, ApiState<TrialDetails>>>({});
  const [ activeCheckpoint, setActiveCheckpoint ] = useState<CheckpointWorkloadExtended>();
  const [ showCheckpoint, setShowCheckpoint ] = useState(false);
  const [ source ] = useState(axios.CancelToken.source());
  const [ canceler ] = useState(new AbortController());

  const fetchTrialDetails = useCallback(async (trialId) => {
    try {
      const response = await getTrialDetails({ id: trialId }, { signal: canceler.signal });
      setTrialsDetails(prev => (
        { ...prev, [trialId]: { ...prev[trialId], data: response, isLoading: false } }
      ));
    } catch (e) {
      if (!isAborted(e)) {
        setTrialsDetails(prev => ({ ...prev, [trialId]: { ...prev[trialId], error: e } }));
      }
    }
  }, [ canceler.signal ]);

  useEffect(() => {
    return () => {
      source.cancel();
      canceler.abort();
    };
  }, [ canceler, source ]);

  useEffect(() => {
    const trialIds = trials.map(trial => {
      setTrialsDetails(prev =>
        ({
          ...prev,
          [trial.id]: {
            data: undefined,
            error: undefined,
            isLoading: true,
            source,
          },
        }));
      return trial.id;
    });
    trialIds.forEach(trialId => fetchTrialDetails(trialId));
  }, [ fetchTrialDetails, source, trials ]);

  const handleCheckpointShow = (
    event: React.MouseEvent,
    trial: TrialItem,
  ) => {
    if (trial.bestAvailableCheckpoint) {
      const checkpoint = {
        ...trial.bestAvailableCheckpoint,
        experimentId: trial.experimentId,
        trialId: trial.id,
      };
      event.stopPropagation();
      setActiveCheckpoint(checkpoint);
      setShowCheckpoint(true);
    }
  };

  const handleCheckpointDismiss = useCallback(() => setShowCheckpoint(false), []);

  const handleTrialUnselect = useCallback((trial: TrialItem) =>
    onUnselect(trial.id), [ onUnselect ]);

  const durations: Record<string, TrialDurations> = useMemo(
    () => Object.fromEntries(Object.values(trialsDetails)
      .map(trial => (trial.data ? [ trial.data.id, trialDurations(trial.data.workloads) ] : [])))
    , [ trialsDetails ],
  );

  const getCheckpointSize = useCallback((trial: TrialDetails) => {
    const totalBytes = trial.workloads
      .filter(step => step.checkpoint
      && step.checkpoint.state === CheckpointState.Completed)
      .map(step => checkpointSize(step.checkpoint as CheckpointWorkload))
      .reduce((acc, cur) => acc + cur, 0);
    return humanReadableBytes(totalBytes);
  }, []);

  const totalCheckpointsSizes: Record<string, string> = useMemo(
    () => Object.fromEntries(Object.values(trialsDetails)
      .map(trial => trial.data ? [ trial.data.id, getCheckpointSize(trial.data) ] : []))
    , [ getCheckpointSize, trialsDetails ],
  );

  const metricNames = useMemo(() => {
    const nameSet: Record<string, MetricName> = {};
    trials.forEach(trial => {
      extractMetricNames(trialsDetails[trial.id]?.data?.workloads || [])
        .forEach(item => nameSet[item.name] = (item));
    });
    return Object.values(nameSet);
  }, [ trialsDetails, trials ]);

  const hyperparameterNames = useMemo(
    () =>
      Object.keys(trials.first().hyperparameters),
    [ trials ],
  );

  console.log(trialsDetails);

  return (
    <>
      <div className={css.tableContainer}>
        <div
          className={css.headerRow}>
          <div />
          {trials.map(trial =>
            <Tag
              className={[ css.trialTag, css.centerVertically ].join(' ')}
              closable
              key={trial.id}
              onClose={() => handleTrialUnselect(trial)}>Trial {trial.id}</Tag>)}</div>
        <div className={css.row}>
          <h3>State</h3>
          {trials.map(trial =>
            <div className={css.centerVertically} key={trial.id}>
              <Badge state={trial.state} type={BadgeType.State} />
            </div>)}
        </div>
        <div className={css.row}>
          <h3>Start Time</h3>
          {trials.map(trial =>
            <p key={trial.id}>
              {shortEnglishHumannizer(getDuration({ startTime: trial.startTime }))} ago
            </p>)}
        </div>
        <div className={css.row}>
          <h3>Training Time</h3>
          {trials.map(trial =>
            <p key={trial.id}>
              {shortEnglishHumannizer(durations[trial.id]?.train)}
            </p>)}
        </div>
        <div className={css.row}>
          <h3>Validation Time</h3>
          {trials.map(trial =>
            <p key={trial.id}>
              {shortEnglishHumannizer(durations[trial.id]?.validation)}
            </p>)}
        </div>
        <div className={css.row}>
          <h3>Checkpoint Time</h3>
          {trials.map(trial =>
            <p key={trial.id}>
              {shortEnglishHumannizer(durations[trial.id]?.checkpoint)}
            </p>)}
        </div>
        <div className={css.row}>
          <h3>Batches Processed</h3>
          {trials.map(trial => <p key={trial.id}>{trial.totalBatchesProcessed}</p>)}
        </div>
        <div className={css.row}>
          <h3>Best Checkpoint</h3>
          {trials.map(trial =>
            trial.bestAvailableCheckpoint ?
              <Button
                className={css.checkpointButton}
                key={trial.id}
                onClick={e => handleCheckpointShow(e, trial)}>
                <Icon name="checkpoint" />
                <span>Batch {trial.bestAvailableCheckpoint?.totalBatches}</span>
              </Button> : <div />)}
        </div>
        <div className={css.row}>
          <h3>Total Checkpoint Size</h3>
          {trials.map(trial => <p key={trial.id}>{totalCheckpointsSizes[trial.id]}</p>)}
        </div>
        <div className={css.headerRow}><h2>Metrics</h2></div>
        {metricNames.map(metric =>
          <div className={css.row} key={metric.name}>
            <BadgeTag label={metric.name}>{metric.type === MetricType.Training ?
              <Tooltip title="training">T</Tooltip> :
              <Tooltip title="validation">V</Tooltip>}
            </BadgeTag>
          </div>)}
        <div className={css.headerRow}><h2>Hyperparameters</h2></div>
        {hyperparameterNames.map(hp =>
          <div className={css.row} key={hp}>
            <h3>{hp}</h3>
            {trials.map(trial =>
              !isNaN(parseFloat(JSON.stringify(trial.hyperparameters[hp]))) ?
                <HumanReadableFloat
                  key={trial.id}
                  num={parseFloat(JSON.stringify(trial.hyperparameters[hp]))} />:
                trial.hyperparameters[hp])}
          </div>)}
      </div>
      {activeCheckpoint && <CheckpointModal
        checkpoint={activeCheckpoint}
        config={experiment.config}
        show={showCheckpoint}
        title={`Best Checkpoint for Trial ${activeCheckpoint.trialId}`}
        onHide={handleCheckpointDismiss} />}
    </>
  );
};

export default TrialsComparisonModal;
