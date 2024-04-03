import { ModalCloseReason, useModal } from 'hew/Modal';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { isEqual } from 'lodash';
import { useCallback, useEffect, useState } from 'react';

import CheckpointModalComponent from 'components/CheckpointModal';
import ModelCreateModal from 'components/ModelCreateModal';
import RegisterCheckpointModal from 'components/RegisterCheckpointModal';
import { getModels } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import {
  CheckpointWorkloadExtended,
  CoreApiGenericCheckpoint,
  ExperimentConfig,
  ModelItem,
} from 'types';
import handleError, { ErrorType } from 'utils/error';
import { validateDetApiEnum } from 'utils/service';

interface Return {
  checkpointModalComponent: React.ReactNode;
  openCheckpoint: () => void;
  modelCreateModalComponent: React.ReactNode;
  registerModalComponent: React.ReactNode;
}

export const useCheckpointFlow = ({
  checkpoint,
  config,
  title,
  models: modelsIn = NotLoaded,
}: {
  checkpoint?: CheckpointWorkloadExtended | CoreApiGenericCheckpoint;
  config: ExperimentConfig;
  title: string;
  models?: Loadable<ModelItem[]>;
}): Return => {
  const modelCreateModal = useModal(ModelCreateModal);
  const checkpointModal = useModal(CheckpointModalComponent);
  const registerModal = useModal(RegisterCheckpointModal);

  const [models, setModels] = useState<Loadable<ModelItem[]>>(modelsIn);
  const [selectedModelName, setSelectedModelName] = useState<string>();
  const [canceler] = useState(new AbortController());

  const openCheckpoint = useCallback(() => {
    checkpointModal.open();
  }, [checkpointModal]);

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
    if (models.isNotLoaded) fetchModels();
  }, [fetchModels, models.isNotLoaded]);

  return {
    checkpointModalComponent: (
      <checkpointModal.Component
        checkpoint={checkpoint}
        config={config}
        title={title}
        onClose={(reason?: ModalCloseReason) => {
          if (reason === 'Ok') registerModal.open();
        }}
      />
    ),
    modelCreateModalComponent: <modelCreateModal.Component onClose={handleOnCloseCreateModel} />,
    openCheckpoint,
    registerModalComponent: (
      <registerModal.Component
        checkpoints={checkpoint?.uuid ?? []}
        closeModal={registerModal.close}
        modelName={selectedModelName}
        models={models}
        openModelModal={modelCreateModal.open}
      />
    ),
  };
};
