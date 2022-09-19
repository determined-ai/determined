import { ModalFuncProps } from 'antd/es/modal/Modal';
import { useCallback, useEffect, useState } from 'react';

import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { deleteModelVersion } from 'services/api';
import useModal, {
  CANNOT_DELETE_MODAL_PROPS,
  ModalHooks as Hooks,
  ModalCloseReason,
} from 'shared/hooks/useModal/useModal';
import { clone } from 'shared/utils/data';
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
  const [modelVersion, setModelVersion] = useState<ModelVersion>();

  const { canDeleteModelVersion } = usePermissions();

  const handleOnClose = useCallback(() => {
    setModelVersion(undefined);
    onClose?.(ModalCloseReason.Cancel);
  }, [onClose]);

  const { modalOpen: openOrUpdate, ...modalHook } = useModal({ onClose: handleOnClose });

  const handleOk = useCallback(async () => {
    if (!modelVersion) return Promise.reject();

    try {
      await deleteModelVersion({
        modelName: modelVersion.model.name ?? '',
        versionId: modelVersion.id ?? 0,
      });
      routeToReactUrl(paths.modelDetails(String(modelVersion.model.id)));
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: `Unable to delete model version ${modelVersion.id}.`,
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [modelVersion]);

  const getModalProps = useCallback(
    (modelVersion: ModelVersion): ModalFuncProps => {
      return canDeleteModelVersion({ modelVersion })
        ? {
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
            onOk: handleOk,
            title: 'Confirm Delete',
          }
        : clone(CANNOT_DELETE_MODAL_PROPS);
    },
    [canDeleteModelVersion, handleOk],
  );

  const modalOpen = useCallback((modelVersion: ModelVersion) => setModelVersion(modelVersion), []);

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modelVersion) openOrUpdate(getModalProps(modelVersion));
  }, [getModalProps, modelVersion, openOrUpdate]);

  return { modalOpen, ...modalHook };
};

export default useModalModelVersionDelete;
