import { ComponentStory } from '@storybook/react';
import React from 'react';

import Label from './Label';

export default {
  component: Label,
  title: 'Determined/Label',
};

export const Default: ComponentStory<typeof Label> = (args) => <Label {...args} />;

Default.args = { children: 'Default Label' };
