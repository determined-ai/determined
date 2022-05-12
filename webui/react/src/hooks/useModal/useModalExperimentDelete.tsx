import { ExclamationCircleOutlined } from '@ant-design/icons';
import { ModalFuncProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import { paths } from 'routes/utils';
import { deleteExperiment } from 'services/api';
import handleError from 'utils/error';

import { ErrorLevel, ErrorType } from '../../shared/utils/error';
import { routeToReactUrl } from '../../shared/utils/routes';

import useModal, { ModalHooks } from './useModal';

interface Props {
  experimentId: number;
  onClose?: () => void;
}

const useModalExperimentDelete = ({ experimentId, onClose }: Props): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal({ onClose });

  const handleOk = useCallback(async () => {
    try {
      await deleteExperiment({ experimentId: experimentId });
      routeToReactUrl(paths.experimentList());
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete experiment.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ experimentId ]);

  const modalProps: ModalFuncProps = useMemo(() => {
    return {
      content: `Are you sure you want to delete\n experiment ${experimentId}?`,
      icon: <ExclamationCircleOutlined />,
      okText: 'Delete',
      onOk: handleOk,
      title: 'Confirm Experiment Deletion',
    };
  }, [ handleOk, experimentId ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    openOrUpdate({ ...modalProps, ...initialModalProps });
  }, [ modalProps, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalExperimentDelete;
