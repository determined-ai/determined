import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import Toggle from './Toggle';

export default {
  component: Toggle,
  title: 'Determined/Toggle',
} as Meta<typeof Toggle>;

export const Default: ComponentStory<typeof Toggle> = (args) => <Toggle {...args} />;

Default.args = { prefixLabel: 'Prefix Label', suffixLabel: '' };
