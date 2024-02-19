import Card from 'hew/Card';
import { useModal } from 'hew/Modal';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { isEqual } from 'lodash';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import CheckpointModalComponent from 'components/CheckpointModal';
import ModelCreateModal from 'components/ModelCreateModal';
import OverviewStats from 'components/OverviewStats';
import RegisterCheckpointModal from 'components/RegisterCheckpointModal';
import ResourceAllocationModalComponent from 'components/ResourceAllocationModal';
import Section from 'components/Section';
import TimeAgo from 'components/TimeAgo';
import { getModels, getTaskAllocation } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import clusterStore from 'stores/cluster';
import {
  AllocationData,
  CheckpointWorkloadExtended,
  ExperimentBase,
  ModelItem,
  Resource,
  ResourcePool,
  RunState,
  TrialDetails,
} from 'types';
import handleError, { ErrorType } from 'utils/error';
import { validateDetApiEnum } from 'utils/service';
import { humanReadableBytes } from 'utils/string';

import css from './TrialInfoBox.module.scss';

interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

interface RpData extends ResourcePool {
  isRPFull: boolean;
  isRunning: boolean;
  name: string;
  nodes: Array<{ nodeName: string; slotsIds: Resource[] }>;
  totalSlots: number;
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
  const [taskAllocation, setTaskAllocation] =
    useState<Loadable<AllocationData | undefined>>(NotLoaded);
  const [selectedModelName, setSelectedModelName] = useState<string>();
  const shouldRenderAllocationCard = useMemo(
    () => trial !== undefined && experiment.numTrials === 1,
    [trial, experiment],
  ); // as per ticket requirements, we're only rendering it on single trial experiments and trial details pages
  const resourcePools = Loadable.getOrElse([], useObservable(clusterStore.resourcePools));

  const experimentRPInfo = useMemo(() => {
    if (!trial) return undefined;

    const rpLabel = experiment.resourcePool;
    const rpData = resourcePools.find((rp) => rp.name === rpLabel);

    if (rpData === undefined) return undefined;

    const allocation = Loadable.getOrElse(undefined, taskAllocation);
    const rpUsedSlots =
      allocation?.acceleratorData.reduce((acc, nodes) => {
        if (nodes.acceleratorUuids) acc = acc + nodes.acceleratorUuids.length;
        return acc;
      }, 0) || 0;

    if (allocation === undefined) return undefined;

    const getSlots = (accelerators: string[]) => {
      const slots = accelerators.map((slotId) => ({
        container: { id: slotId },
      })) as Resource[];

      if (rpData.slotsPerAgent !== undefined && slots.length < rpData.slotsPerAgent) {
        for (let i = 0; i < rpData.slotsPerAgent; i++) {
          slots.push({ container: undefined } as Resource);
        }
      }

      return slots;
    };

    return {
      ...rpData,
      isRPFull: rpUsedSlots === (experiment.config.resources.slots_per_trial || 0),
      isRunning: trial.state === RunState.Active,
      name: rpData.name || experiment.resourcePool,
      nodes: allocation.acceleratorData.map((node) => ({
        nodeName: node.nodeName || '',
        slotsIds: getSlots(node.acceleratorUuids || []),
      })),
      slotsUsed: rpUsedSlots,
      totalSlots: (rpData.slotsPerAgent || 1) * rpData.maxAgents,
    } as RpData;
  }, [experiment, resourcePools, trial, taskAllocation]);

  const modelCreateModal = useModal(ModelCreateModal);
  const checkpointModal = useModal(CheckpointModalComponent);
  const registerModal = useModal(RegisterCheckpointModal);
  const allocationModal = useModal(ResourceAllocationModalComponent);

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

  const fetchTaskAllocation = useCallback(async () => {
    if (!trial) return;

    // one big issue is that taskIds is an optional property
    const taskId = trial.taskIds !== undefined ? trial.taskIds[trial.taskIds.length - 1] : '';

    if (!taskId) return;

    try {
      const response = await getTaskAllocation(taskId, {
        signal: canceler.signal,
      });

      setTaskAllocation((prev) => {
        const loadedAllocation = Loaded(response);

        if (isEqual(prev, loadedAllocation)) return prev;

        return loadedAllocation;
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to fetch task allocation data.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [canceler.signal, trial]);

  useEffect(() => {
    fetchModels();
    fetchTaskAllocation();
  }, [fetchModels, fetchTaskAllocation]);

  const handleModalCheckpointClick = useCallback(() => {
    checkpointModal.open();
  }, [checkpointModal]);

  const handleModalAllocationClick = useCallback(() => {
    allocationModal.open();
  }, [allocationModal]);

  const handleAllocationModalClose = useCallback(
    () => allocationModal.close(''),
    [allocationModal],
  );

  const appendText = (n: number) => `Slot${n > 1 ? 's' : ''}`;

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
              <span className={css.modalLink}>Batch {bestCheckpoint.totalBatches}</span>
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
        {shouldRenderAllocationCard && experimentRPInfo !== undefined && (
          <>
            <OverviewStats title="Resource Allocation" onClick={handleModalAllocationClick}>
              <span className={css.modalLink}>
                {experimentRPInfo.isRPFull
                  ? `${experimentRPInfo.slotsUsed} ${appendText(experimentRPInfo.slotsUsed)}`
                  : `${experimentRPInfo.slotsUsed}/${experimentRPInfo.totalSlots} ${appendText(
                      experimentRPInfo.totalSlots,
                    )}`}
              </span>
            </OverviewStats>
            <allocationModal.Component
              accelerator={experimentRPInfo.accelerator || ''}
              isRunning={experimentRPInfo.isRunning}
              nodes={experimentRPInfo.nodes}
              rpName={experimentRPInfo.name}
              onClose={handleAllocationModalClose}
            />
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
