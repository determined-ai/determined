import Card from 'hew/Card';
import { useModal } from 'hew/Modal';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { isEqual } from 'lodash';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import CheckpointModalComponent from 'components/CheckpointModal';
import ModelCreateModal from 'components/ModelCreateModal';
import OverviewStats from 'components/OverviewStats';
import RegisterCheckpointModal from 'components/RegisterCheckpointModal';
import Section from 'components/Section';
import TimeAgo from 'components/TimeAgo';
import { getModels } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import { CheckpointWorkloadExtended, ExperimentBase, ModelItem, TrialDetails } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { validateDetApiEnum } from 'utils/service';
import { humanReadableBytes } from 'utils/string';

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

  const [canceler] = useState(new AbortController());
  const [models, setModels] = useState<Loadable<ModelItem[]>>(NotLoaded);
  const [selectedModelName, setSelectedModelName] = useState<string>();

  const modelCreateModal = useModal(ModelCreateModal);
  const checkpointModal = useModal(CheckpointModalComponent);
  const registerModal = useModal(RegisterCheckpointModal);

  const handleOnCloseCreateModel = useCallback(
    (modelName?: string) => {
      if (modelName) {
        setSelectedModelName(modelName);
        registerModal.open();
      }
    },
    [setSelectedModelName, registerModal],
  );

  const fetchModels = useCallback(async () => {
    try {
      const response = await getModels(
        {
          archived: false,
          orderBy: 'ORDER_BY_DESC',
          sortBy: validateDetApiEnum(
            V1GetModelsRequestSortBy,
            V1GetModelsRequestSortBy.LASTUPDATEDTIME,
          ),
        },
        { signal: canceler.signal },
      );
      setModels((prev) => {
        const loadedModels = Loaded(response.models);
        if (isEqual(prev, loadedModels)) return prev;
        return loadedModels;
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to fetch models.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [canceler.signal]);

  useEffect(() => {
    fetchModels();
  }, [fetchModels]);

  const handleModalCheckpointClick = useCallback(() => {
    checkpointModal.open();
  }, [checkpointModal]);

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
            <OverviewStats title="Best Checkpoint!" onClick={handleModalCheckpointClick}>
              Batch {bestCheckpoint.totalBatches}
            </OverviewStats>
            <registerModal.Component
              checkpoints={bestCheckpoint.uuid ? [bestCheckpoint.uuid] : []}
              closeModal={() => registerModal.close('ok')}
              modelName={selectedModelName}
              models={models}
              openModelModal={modelCreateModal.open}
            />
            <checkpointModal.Component
              checkpoint={bestCheckpoint}
              config={experiment.config}
              title="Best Checkpoint"
            />
            <modelCreateModal.Component onClose={handleOnCloseCreateModel} />
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
            {`${experiment.checkpoints} (${checkpointsSize})`}
          </OverviewStats>
        )}
      </Card.Group>
    </Section>
  );
};
