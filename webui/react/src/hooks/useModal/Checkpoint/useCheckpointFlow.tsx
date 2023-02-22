import { ReactElement, useCallback, useMemo } from 'react';

import { ModalCloseReason } from 'shared/hooks/useModal/useModal';
import { CheckpointWorkloadExtended, CoreApiGenericCheckpoint, ExperimentConfig } from 'types';

import useModalModelCreate from '../Model/useModalModelCreate';

import useModalCheckpoint from './useModalCheckpoint';
import useModalCheckpointRegister from './useModalCheckpointRegister';

interface Return {
  contextHolder: ReactElement[];
  openCheckpoint: () => void;
}

export const useCheckpointFlow = ({
  checkpoint,
  config,
  title,
}: {
  checkpoint?: CheckpointWorkloadExtended | CoreApiGenericCheckpoint;
  config: ExperimentConfig;
  title: string;
}): Return => {
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
      if (reason === ModalCloseReason.Ok && checkpoint?.uuid) {
        openModalCheckpointRegister({ checkpoints: checkpoint.uuid });
      }
    },
    [checkpoint, openModalCheckpointRegister],
  );

  const { contextHolder: modalCheckpointContextHolder, modalOpen: openModalCheckpoint } =
    useModalCheckpoint({
      checkpoint,
      config,
      onClose: handleOnCloseCheckpoint,
      title,
    });

  const openCheckpoint = useCallback(() => {
    openModalCheckpoint();
  }, [openModalCheckpoint]);

  const contextHolder = useMemo(
    () => [
      modalCheckpointContextHolder,
      modalCheckpointRegisterContextHolder,
      modalModelCreateContextHolder,
    ],
    [
      modalCheckpointRegisterContextHolder,
      modalModelCreateContextHolder,
      modalCheckpointContextHolder,
    ],
  );

  return {
    contextHolder,
    openCheckpoint,
  };
};
