import React, { useCallback, useMemo } from 'react';

import Card from 'components/kit/Card';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import TimeAgo from 'components/TimeAgo';
import useModalCheckpoint from 'hooks/useModal/Checkpoint/useModalCheckpoint';
import useModalCheckpointRegister from 'hooks/useModal/Checkpoint/useModalCheckpointRegister';
import useModalModelCreate from 'hooks/useModal/Model/useModalModelCreate';
import { ModalCloseReason } from 'shared/hooks/useModal/useModal';
import { humanReadableBytes } from 'shared/utils/string';
import { CheckpointWorkloadExtended, ExperimentBase, TrialDetails } from 'types';

interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

const TrialInfoBox: React.FC<Props> = ({ trial, experiment }: Props) => {
  const bestCheckpoint: CheckpointWorkloadExtended | undefined = useMemo(() => {
    if (!trial) return;
    const cp = trial.bestAvailableCheckpoint;
    if (!cp) return;

    return {
      ...cp,
      experimentId: trial.experimentId,
      trialId: trial.id,
    };
  }, [trial]);

  const totalCheckpointsSize = useMemo(() => {
    const totalBytes = trial?.totalCheckpointSize;
    if (!totalBytes) return;
    return humanReadableBytes(totalBytes);
  }, [trial?.totalCheckpointSize]);

  const {
    contextHolder: modalCheckpointRegisterContextHolder,
    modalOpen: openModalCheckpointRegister,
  } = useModalCheckpointRegister({
    onClose: (reason?: ModalCloseReason, checkpoints?: string[]) => {
      if (checkpoints) openModalCreateModel({ checkpoints });
    },
  });

  const handleOnCloseCreateModel = useCallback(
    (reason?: ModalCloseReason, checkpoints?: string[], modelName?: string) => {
      if (checkpoints) openModalCheckpointRegister({ checkpoints, selectedModelName: modelName });
    },
    [openModalCheckpointRegister],
  );

  const { contextHolder: modalModelCreateContextHolder, modalOpen: openModalCreateModel } =
    useModalModelCreate({ onClose: handleOnCloseCreateModel });

  const handleOnCloseCheckpoint = useCallback(
    (reason?: ModalCloseReason) => {
      if (reason === ModalCloseReason.Ok && bestCheckpoint?.uuid) {
        openModalCheckpointRegister({ checkpoints: bestCheckpoint.uuid });
      }
    },
    [bestCheckpoint, openModalCheckpointRegister],
  );

  const { contextHolder: modalCheckpointContextHolder, modalOpen: openModalCheckpoint } =
    useModalCheckpoint({
      checkpoint: bestCheckpoint,
      config: experiment.config,
      onClose: handleOnCloseCheckpoint,
      title: 'Best Checkpoint',
    });

  const handleModalCheckpointClick = useCallback(() => {
    openModalCheckpoint();
  }, [openModalCheckpoint]);

  return (
    <Section>
      <Card.Group size="small">
        {trial?.runnerState && (
          <OverviewStats title="Last Runner State">{trial.runnerState}</OverviewStats>
        )}
        {trial?.startTime && (
          <OverviewStats title="Started">
            <TimeAgo datetime={trial.startTime} />
          </OverviewStats>
        )}
        {totalCheckpointsSize && (
          <OverviewStats title="Checkpoints">{`${trial?.checkpointCount} (${totalCheckpointsSize})`}</OverviewStats>
        )}
        {bestCheckpoint && (
          <>
            <OverviewStats title="Best Checkpoint" onClick={handleModalCheckpointClick}>
              Batch {bestCheckpoint.totalBatches}
            </OverviewStats>
            {modalCheckpointContextHolder}
            {modalCheckpointRegisterContextHolder}
            {modalModelCreateContextHolder}
          </>
        )}
      </Card.Group>
    </Section>
  );
};

export default TrialInfoBox;

export const TrialInfoBoxMultiTrial: React.FC<Props> = ({ experiment }: Props) => {
  const searcher = experiment.config.searcher;
  const checkpointsSize = useMemo(() => {
    const totalBytes = experiment?.checkpointSize;
    if (!totalBytes) return;
    return humanReadableBytes(totalBytes);
  }, [experiment]);
  return (
    <Section>
      <Card.Group size="small">
        {searcher?.metric && <OverviewStats title="Metric">{searcher.metric}</OverviewStats>}
        {searcher?.name && <OverviewStats title="Searcher">{searcher.name}</OverviewStats>}
        {experiment.numTrials > 0 && (
          <OverviewStats title="Trials">{experiment.numTrials}</OverviewStats>
        )}
        {checkpointsSize && (
          <OverviewStats title="Checkpoints">
            {`${experiment.checkpointCount} (${checkpointsSize})`}
          </OverviewStats>
        )}
      </Card.Group>
    </Section>
  );
};
