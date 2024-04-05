import { ModalCloseReason, useModal } from 'hew/Modal';
import { Loadable } from 'hew/utils/loadable';
import { useCallback, useState } from 'react';

import CheckpointModalComponent from 'components/CheckpointModal';
import ModelCreateModal from 'components/ModelCreateModal';
import RegisterCheckpointModal from 'components/RegisterCheckpointModal';
import {
  CheckpointWorkloadExtended,
  CoreApiGenericCheckpoint,
  ExperimentConfig,
  ModelItem,
} from 'types';

import { useFetchModels } from './useFetchModels';

interface Return {
  checkpointModalComponents: React.ReactNode;
  openCheckpoint: () => void;
}

export const useCheckpointFlow = ({
  checkpoint,
  config,
  title,
  models: modelsIn,
}: {
  checkpoint?: CheckpointWorkloadExtended | CoreApiGenericCheckpoint;
  config: ExperimentConfig;
  title: string;
  models?: Loadable<ModelItem[]>;
}): Return => {
  const modelCreateModal = useModal(ModelCreateModal);
  const checkpointModal = useModal(CheckpointModalComponent);
  const registerModal = useModal(RegisterCheckpointModal);

  const models = useFetchModels(modelsIn);
  const [selectedModelName, setSelectedModelName] = useState<string>();

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

  return {
    checkpointModalComponents: (
      <>
        <checkpointModal.Component
          checkpoint={checkpoint}
          config={config}
          title={title}
          onClose={(reason?: ModalCloseReason) => {
            if (reason === 'Ok') registerModal.open();
          }}
        />
        <modelCreateModal.Component onClose={handleOnCloseCreateModel} />
        <registerModal.Component
          checkpoints={checkpoint?.uuid ?? []}
          closeModal={registerModal.close}
          modelName={selectedModelName}
          models={models}
          openModelModal={modelCreateModal.open}
        />
      </>
    ),
    openCheckpoint,
  };
};
