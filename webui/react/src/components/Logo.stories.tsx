import { ComponentMeta, ComponentStory } from '@storybook/react';
import React from 'react';

import { BrandingType } from 'types';

import Logo, { Orientation } from './Logo';

export default {
  argTypes: {
    branding: { control: 'inline-radio', options: BrandingType },
    orientation: { control: 'inline-radio', options: Orientation },
  },
  component: Logo,
  title: 'Determined/Logo',
} as ComponentMeta<typeof Logo>;

export const Default: ComponentStory<typeof Logo> = (args) => <Logo {...args} />;

Default.args = {
  branding: BrandingType.Determined,
  orientation: Orientation.Horizontal,
};
