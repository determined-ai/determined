import { ModalFuncProps } from 'antd/es/modal/Modal';
import { useCallback } from 'react';

import { paths } from 'routes/utils';
import { deleteModelVersion } from 'services/api';
import useModal, { ModalHooks as Hooks, ModalCloseReason } from 'shared/hooks/useModal/useModal';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { ModelVersion } from 'types';
import handleError from 'utils/error';

interface Props {
  onClose?: (reason?: ModalCloseReason) => void;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (modelVersion: ModelVersion) => void;
}

const useModalModelVersionDelete = ({ onClose }: Props = {}): ModalHooks => {
  const handleOnClose = useCallback(() => {
    onClose?.(ModalCloseReason.Cancel);
  }, [onClose]);

  const { modalOpen: openOrUpdate, ...modalHook } = useModal({ onClose: handleOnClose });

  const getModalProps = useCallback((modelVersion: ModelVersion): ModalFuncProps => {
    return {
      closable: true,
      content: `
        Are you sure you want to delete this version
        "Version ${modelVersion.version}" from this model?
      `,
      icon: null,
      maskClosable: true,
      okButtonProps: { type: 'primary' },
      okText: 'Delete Version',
      okType: 'danger',
      onOk: async () => {
        if (!modelVersion) return Promise.reject();

        try {
          await deleteModelVersion({
            modelName: modelVersion.model.name ?? '',
            versionNum: modelVersion.version ?? 0,
          });
          routeToReactUrl(paths.modelDetails(String(modelVersion.model.id)));
        } catch (e) {
          handleError(e, {
            level: ErrorLevel.Error,
            publicMessage: 'Please try again later.',
            publicSubject: `Unable to delete model version ${modelVersion.version}.`,
            silent: false,
            type: ErrorType.Server,
          });
        }
      },
      title: 'Confirm Delete',
    };
  }, []);

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  const modalOpen = useCallback(
    (modelVersion: ModelVersion) => {
      openOrUpdate(getModalProps(modelVersion));
    },
    [getModalProps, openOrUpdate],
  );

  return { modalOpen, ...modalHook };
};

export default useModalModelVersionDelete;
