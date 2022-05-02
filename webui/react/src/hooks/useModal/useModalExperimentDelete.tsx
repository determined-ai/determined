import { ExclamationCircleOutlined } from '@ant-design/icons';
import { ModalFuncProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import { paths, routeToReactUrl } from 'routes/utils';
import { deleteExperiment } from 'services/api';
import { ExperimentBase } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

import useModal, { ModalHooks } from './useModal';

interface Props {
  experiment: ExperimentBase;
  onClose?: () => void;
}

const useModalExperimentDelete = ({ experiment, onClose }: Props): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal({ onClose });

  const handleOk = useCallback(async () => {
    try {
      await deleteExperiment({ experimentId: experiment.id });
      routeToReactUrl(paths.projectDetails(experiment.projectId));
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete experiment.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ experiment.id, experiment.projectId ]);

  const modalProps: ModalFuncProps = useMemo(() => {
    return {
      content: 'Are you sure you want to delete\nthis experiment?',
      icon: <ExclamationCircleOutlined />,
      okText: 'Delete',
      onOk: handleOk,
      title: 'Confirm Experiment Deletion',
    };
  }, [ handleOk ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    openOrUpdate({ ...modalProps, ...initialModalProps });
  }, [ modalProps, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalExperimentDelete;
