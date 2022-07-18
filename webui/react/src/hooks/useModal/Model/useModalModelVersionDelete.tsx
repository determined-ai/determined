import { ModalFuncProps } from 'antd/es/modal/Modal';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';
import { deleteModelVersion } from 'services/api';
import { clone } from 'shared/utils/data';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { ModelVersion } from 'types';
import handleError from 'utils/error';

import useModal, {
  CANNOT_DELETE_MODAL_PROPS, ModalHooks as Hooks, ModalCloseReason,
} from '../../../shared/hooks/useModal/useModal';

interface Props {
  onClose?: (reason?: ModalCloseReason) => void;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (modelVersion: ModelVersion) => void;
}

const useModalModelVersionDelete = ({ onClose }: Props = {}): ModalHooks => {
  const { auth: { user } } = useStore();
  const [ modelVersion, setModelVersion ] = useState<ModelVersion>();

  const isDeletable = useMemo(() => {
    return user?.isAdmin || user?.id === modelVersion?.userId;
  }, [ user, modelVersion ]);

  const handleOnClose = useCallback(() => {
    setModelVersion(undefined);
    onClose?.(ModalCloseReason.Cancel);
  }, [ onClose ]);

  const { modalOpen: openOrUpdate, ...modalHook } = useModal({ onClose: handleOnClose });

  const handleOk = useCallback(async () => {
    if (!modelVersion) return Promise.reject();

    try {
      await deleteModelVersion({
        modelName: modelVersion.model.name ?? '',
        versionId: modelVersion.id ?? 0,
      });
      routeToReactUrl(paths.modelList());
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: `Unable to delete model version ${modelVersion.id}.`,
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ modelVersion ]);

  const getModalProps = useCallback((modelVersion: ModelVersion): ModalFuncProps => {
    return isDeletable ? {
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
    } : clone(CANNOT_DELETE_MODAL_PROPS);
  }, [ handleOk, isDeletable ]);

  const modalOpen = useCallback((modelVersion: ModelVersion) => setModelVersion(modelVersion), []);

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modelVersion) openOrUpdate(getModalProps(modelVersion));
  }, [ getModalProps, modelVersion, openOrUpdate ]);

  return { modalOpen, ...modalHook };
};

export default useModalModelVersionDelete;
