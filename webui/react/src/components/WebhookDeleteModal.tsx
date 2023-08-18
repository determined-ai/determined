import React from 'react';

import { Modal } from 'components/kit/Modal';
import { paths } from 'routes/utils';
import { deleteWebhook } from 'services/api';
import { Webhook } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { routeToReactUrl } from 'utils/routes';

interface Props {
  webhook?: Webhook;
}

const WebhookDeleteModalComponent: React.FC<Props> = ({ webhook }: Props) => {
  const handleSubmit = async () => {
    if (!webhook) return;
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

  return (
    <Modal
      cancel
      danger
      size="small"
      submit={{
        handleError,
        handler: handleSubmit,
        text: 'Delete Webhook',
      }}
      title="Confirm Delete">
      Are you sure you want to delete this webhook?
    </Modal>
  );
};

export default WebhookDeleteModalComponent;
