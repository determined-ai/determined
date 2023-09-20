import React from 'react';

import { message, notification } from 'components/kit/internal/dialogApi';

import Icon from './Icon';

export type Severity = 'Info' | 'Confirm' | 'Warning' | 'Error';

type NotificationArgs = {
  compact?: never;
  title: string;
  severity?: Severity;
  description: string;
  link?: React.ReactNode;
  closeable?: boolean;
};

type CompactNotificationArgs = {
  compact: true;
  title: string;
  severity?: Severity;
};

export type ToastArgs = NotificationArgs | CompactNotificationArgs;

export const makeToast = (toastArgs: ToastArgs): void => {
  const { compact = false, title, severity = 'Info' } = toastArgs;
  if (compact) {
    switch (severity) {
      case 'Info':
        message.info(title);
        return;
      case 'Confirm':
        message.success(title);
        return;
      case 'Warning':
        message.warning(title);
        return;
      case 'Error':
        message.error(title);
        return;
    }
  } else {
    const { description, link, closeable = false } = toastArgs as NotificationArgs;
    const args = {
      closeIcon: closeable ? <Icon decorative name="close-small" /> : null,
      description: link ? (
        <div>
          <p>{description}</p>
          {link}
        </div>
      ) : (
        description
      ),
      message: title,
    };
    switch (severity) {
      case 'Info':
        notification.open(args);
        return;
      case 'Confirm':
        notification.success(args);
        return;
      case 'Warning':
        notification.warning(args);
        return;
      case 'Error':
        notification.error(args);
        return;
    }
  }
};
