import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import IconButton from './IconButton';

export default {
  component: IconButton,
  title: 'IconButton',
} as Meta<typeof IconButton>;

export const Default = (): React.ReactNode => <IconButton icon="checkmark" label="Okay" />;
export const Custom: ComponentStory<typeof IconButton> = (args) => (
  <IconButton {...args} />
);

Custom.args = {
  icon: 'experiment',
  label: 'Experiment',
};
