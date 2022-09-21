import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import Message, { MessageType } from './Message';

export default {
  component: Message,
  title: 'Shared/Message',
} as Meta<typeof Message>;

export const Default: ComponentStory<typeof Message> = (args) => <Message {...args} />;

export const WarningType = (): React.ReactNode => (
  <Message title="Warning type" type={MessageType.Warning} />
);

export const AlertType = (): React.ReactNode => (
  <Message title="Alert type" type={MessageType.Alert} />
);

export const EmptyType = (): React.ReactNode => (
  <Message title="Empty type" type={MessageType.Empty} />
);

Default.args = {
  message: '',
  title: 'Message title is required',
  type: MessageType.Alert,
};
