import { ModalFuncProps } from 'antd/es/modal/Modal';
import { useCallback } from 'react';

import { paths } from 'routes/utils';
import { deleteWebhook } from 'services/api';
import useModal, { ModalHooks as Hooks, ModalCloseReason } from 'shared/hooks/useModal/useModal';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { Webhook } from 'types';
import handleError from 'utils/error';

interface Props {
  onClose?: (reason?: ModalCloseReason) => void;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (webhook: Webhook) => void;
}

const useModalWebhookDelete = ({ onClose }: Props = {}): ModalHooks => {
  const { modalOpen: openOrUpdate, ...modalHook } = useModal({ onClose });

  const getModalProps = useCallback((webhook: Webhook): ModalFuncProps => {
    const handleOk = async () => {
      try {
        await deleteWebhook({ id: webhook.id });
        routeToReactUrl(paths.webhooks());
      } catch (e) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to delete webhook.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    };

    return {
      closable: true,
      content: 'Are you sure you want to delete this webhook?',
      icon: null,
      okButtonProps: { type: 'primary' },
      okText: 'Delete Webhook',
      okType: 'danger',
      onOk: handleOk,
      title: 'Confirm Delete',
    };
  }, []);

  const modalOpen = useCallback(
    (webhook: Webhook) => {
      openOrUpdate(getModalProps(webhook));
    },
    [getModalProps, openOrUpdate],
  );

  return { modalOpen, ...modalHook };
};

export default useModalWebhookDelete;
