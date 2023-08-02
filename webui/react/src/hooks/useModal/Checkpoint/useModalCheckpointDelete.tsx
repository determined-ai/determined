import { ExclamationCircleOutlined } from '@ant-design/icons';
import { ModalFuncProps } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import useModal, { ModalHooks as Hooks, ModalCloseReason } from 'hooks/useModal/useModal';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import { pluralizer } from 'utils/string';

interface OpenProps {
  checkpoints: string | string[];
  initialModalProps?: ModalFuncProps;
}

export interface Props {
  onClose?: (reason?: ModalCloseReason) => void;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (openProps?: OpenProps) => void;
}

const useModalCheckpointDelete = ({ onClose }: Props): ModalHooks => {
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal();
  const [checkpoints, setCheckpoints] = useState<string | string[]>([]);

  const numCheckpoints = useMemo(() => {
    if (Array.isArray(checkpoints)) return checkpoints.length;
    return 1;
  }, [checkpoints]);

  const handleCancel = useCallback(() => onClose?.(ModalCloseReason.Cancel), [onClose]);

  const handleDelete = useCallback(() => {
    readStream(
      detApi.Checkpoint.deleteCheckpoints({
        checkpointUuids: Array.isArray(checkpoints) ? checkpoints : [checkpoints],
      }),
    );
    onClose?.(ModalCloseReason.Ok);
  }, [checkpoints, onClose]);

  const modalProps: ModalFuncProps = useMemo(() => {
    const content = `Are you sure you want to request deletion for 
${numCheckpoints} ${pluralizer(numCheckpoints, 'checkpoint')}?
This action may complete or fail without further notification.`;

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
  }, [handleCancel, handleDelete, numCheckpoints]);

  const modalOpen = useCallback(
    ({ checkpoints, initialModalProps }: OpenProps = { checkpoints: [] }) => {
      setCheckpoints(checkpoints);
      openOrUpdate({ ...modalProps, ...initialModalProps });
    },
    [modalProps, openOrUpdate],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(modalProps);
  }, [modalProps, modalRef, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalCheckpointDelete;
