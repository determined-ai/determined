import { ExclamationCircleOutlined } from '@ant-design/icons';
import { ModalFuncProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import { paths } from 'routes/utils';
import { deleteExperiment } from 'services/api';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { ExperimentBase } from 'types';
import handleError from 'utils/error';

interface Props {
  experiment: ExperimentBase;
  onClose?: () => void;
}

const useModalExperimentDelete = ({ experiment, onClose }: Props): ModalHooks => {
  const { modalOpen: openOrUpdate, ...modalHook } = useModal({ onClose });

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
  }, [experiment.id, experiment.projectId]);

  const modalProps: ModalFuncProps = useMemo(() => {
    return {
      content: `Are you sure you want to delete\n experiment ${experiment.id}?`,
      icon: <ExclamationCircleOutlined />,
      okText: 'Delete',
      onOk: handleOk,
      title: 'Confirm Experiment Deletion',
    };
  }, [handleOk, experiment.id]);

  const modalOpen = useCallback(
    (initialModalProps: ModalFuncProps = {}) => {
      openOrUpdate({ ...modalProps, ...initialModalProps });
    },
    [modalProps, openOrUpdate],
  );

  return { modalOpen, ...modalHook };
};

export default useModalExperimentDelete;
