import { ModalFuncProps } from 'antd/es/modal/Modal';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';
import { deleteModel } from 'services/api';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { ModelItem } from 'types';
import handleError from 'utils/error';

import useModal, {
  CANNOT_DELETE_MODAL_PROPS, ModalHooks as Hooks, ModalCloseReason,
} from '../useModal';

interface Props {
  onClose?: (reason?: ModalCloseReason) => void;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (model: ModelItem) => void;
}

const useModalModelDelete = ({ onClose }: Props = {}): ModalHooks => {
  const { auth: { user } } = useStore();
  const [ model, setModel ] = useState<ModelItem>();

  const isDeletable = useMemo(() => user?.isAdmin || user?.id === model?.userId, [ user, model ]);

  const handleOnClose = useCallback(() => {
    setModel(undefined);
    onClose?.(ModalCloseReason.Cancel);
  }, [ onClose ]);

  const { modalOpen: openOrUpdate, ...modalHook } = useModal({ onClose: handleOnClose });

  const handleOk = useCallback(async () => {
    if (!model) return Promise.reject();

    try {
      await deleteModel({ modelName: model.name });
      routeToReactUrl(paths.modelList());
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete model.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ model ]);

  const getModalProps = useCallback((model: ModelItem): ModalFuncProps => {
    return isDeletable ? {
      closable: true,
      content: `
        Are you sure you want to delete this model "${model?.name}"
        and all of its versions from the model registry?
      `,
      icon: null,
      maskClosable: true,
      okButtonProps: { type: 'primary' },
      okText: 'Delete Model',
      okType: 'danger',
      onOk: handleOk,
      title: 'Confirm Delete',
    } : CANNOT_DELETE_MODAL_PROPS;
  }, [ handleOk, isDeletable ]);

  const modalOpen = useCallback((model: ModelItem) => setModel(model), []);

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (model) openOrUpdate(getModalProps(model));
  }, [ getModalProps, model, openOrUpdate ]);

  return { modalOpen, ...modalHook };
};

export default useModalModelDelete;
