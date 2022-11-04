import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import Icon from './Icon';

export default {
  argTypes: {
    name: {
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
    size: {
      control: {
        options: [
          'tiny',
          'small',
          'medium',
          'large',
          'big',
          'great',
          'huge',
          'enormous',
          'giant',
          'jumbo',
          'mega',
        ],
        type: 'inline-radio',
      },
    },
  },
  component: Icon,
  title: 'Shared/Icon',
} as Meta<typeof Icon>;

export const Default: ComponentStory<typeof Icon> = (args) => <Icon {...args} />;

Default.args = {
  name: 'dai-logo',
  size: 'small',
  title: '',
};
