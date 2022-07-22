import React from 'react';

import Message, { MessageType } from './Message';

export default {
  component: Message,
  title: 'Message',
};

export const Default = (): React.ReactNode => (
  <Message title="Message title is required" />
);

export const WarningType = (): React.ReactNode => (
  <Message title="Warning type" type={MessageType.Warning} />
);

export const AlertType = (): React.ReactNode => (
  <Message title="Alert type" type={MessageType.Alert} />
);

export const EmptyType = (): React.ReactNode => (
  <Message title="Empty type" type={MessageType.Empty} />
);
