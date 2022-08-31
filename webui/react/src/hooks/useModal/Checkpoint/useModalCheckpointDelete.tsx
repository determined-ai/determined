import { ExclamationCircleOutlined } from '@ant-design/icons';
import { ModalFuncProps } from 'antd';
import React, { useCallback, useEffect, useMemo } from 'react';

import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import useModal, { ModalCloseReason, ModalHooks } from 'shared/hooks/useModal/useModal';
import {
  CheckpointWorkloadExtended,
  CoreApiGenericCheckpoint,
} from 'types';

import css from './useModalCheckpoint.module.scss';

export interface Props {
  checkpoint: CheckpointWorkloadExtended | CoreApiGenericCheckpoint;
  onClose?: (reason?: ModalCloseReason) => void;
}

const useModalCheckpointDelete = ({
  checkpoint,
  onClose,
}: Props): ModalHooks => {
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal();

  const handleCancel = useCallback(() => onClose?.(ModalCloseReason.Cancel), [ onClose ]);

  const handleDelete = useCallback(() => {
    if (!checkpoint.uuid) return;
    readStream(detApi.Checkpoint.deleteCheckpoints({ checkpointUuids: [ checkpoint.uuid ] }));
    onClose?.(ModalCloseReason.Ok);
  }, [ checkpoint ]);

  const modalProps: ModalFuncProps = useMemo(() => {
    const content =
      `Are you sure you want to request checkpoint deletion for batches
${checkpoint.totalBatches}. This action may complete or fail without further notification.`;

    return {
      content,
      icon: <ExclamationCircleOutlined />,
      okButtonProps: { danger: true },
      okText: 'Request Delete',
      onCancel: handleCancel,
      onOk: handleDelete,
      title: 'Confirm Checkpoint Deletion',
      width: 450,
    };
  }, [ checkpoint, handleCancel, handleDelete ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    openOrUpdate({ ...modalProps, ...initialModalProps });
  }, [ modalProps, openOrUpdate ]);

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(modalProps);
  }, [ modalProps, modalRef, openOrUpdate ]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalCheckpointDelete;
