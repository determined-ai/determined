import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import IconButton from './IconButton';

export default {
  argTypes: {
    icon: {
      control: {
        options: [
          'arrow-down',
          'arrow-up',
          'dai-logo',
          'cluster',
          'collapse',
          'command',
          'expand',
          'experiment',
          'grid',
          'jupyter-lab',
          'list',
          'lock',
          'notebook',
          'overflow-horizontal',
          'overflow-vertical',
          'shell',
          'star',
          'tensor-board',
          'tensorflow',
          'user',
          'user-small',
        ],
        type: 'select',
      },
    },
  },
  component: IconButton,
  title: 'Determined/Buttons/IconButton',
} as Meta<typeof IconButton>;

export const Default: ComponentStory<typeof IconButton> = (args) => <IconButton {...args} />;

Default.args = {
  icon: 'experiment',
  iconSize: 'medium',
  label: 'Experiment',
};
