import Button from 'hew/Button';
import Icon from 'hew/Icon';
import { ModalCloseReason, useModal } from 'hew/Modal';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { isEqual } from 'lodash';
import React, { useCallback, useEffect, useState } from 'react';

import ModelCreateModal from 'components/ModelCreateModal';
import RegisterCheckpointModal from 'components/RegisterCheckpointModal';
import { getModels } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import {
  CheckpointWorkloadExtended,
  CoreApiGenericCheckpoint,
  ExperimentBase,
  ModelItem,
} from 'types';
import handleError, { ErrorType } from 'utils/error';
import { validateDetApiEnum } from 'utils/service';

import CheckpointModalComponent from './CheckpointModal';

interface Props {
  checkpoint: CheckpointWorkloadExtended | CoreApiGenericCheckpoint;
  children?: React.ReactNode;
  experiment: ExperimentBase;
  title: string;
}

const CheckpointModalTrigger: React.FC<Props> = ({
  checkpoint,
  experiment,
  title,
  children,
}: Props) => {
  const modelCreateModal = useModal(ModelCreateModal);
  const checkpointModal = useModal(CheckpointModalComponent);

  const registerModal = useModal(RegisterCheckpointModal);

  const [models, setModels] = useState<Loadable<ModelItem[]>>(NotLoaded);
  const [canceler] = useState(new AbortController());

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
    <>
      <span onClick={handleModalCheckpointClick}>
        {children !== undefined ? (
          children
        ) : (
          <Button
            aria-label="View Checkpoint"
            icon={<Icon name="checkpoint" showTooltip title="View Checkpoint" />}
          />
        )}
      </span>
      <registerModal.Component
        checkpoints={checkpoint.uuid ? [checkpoint.uuid] : []}
        closeModal={(reason: ModalCloseReason) => registerModal.close(reason)}
        models={models}
      />
      <modelCreateModal.Component />
      <checkpointModal.Component
        checkpoint={checkpoint}
        config={experiment.config}
        title={title}
        onClose={(reason?: ModalCloseReason) => {
          if (reason === 'Ok') registerModal.open();
        }}
      />
    </>
  );
};

export default CheckpointModalTrigger;
