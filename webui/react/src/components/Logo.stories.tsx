import { select, withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import { enumToOptions } from 'storybook/utils';
import { BrandingType } from 'types';

import Logo, { Orientation } from './Logo';

export default {
  component: Logo,
  decorators: [ withKnobs ],
  parameters: {
    backgrounds: {
      default: 'dark background',
      values: [
        { name: 'dark background', value: '#111' },
      ],
    },
  },
  title: 'Logo',
};

const knobBrandingOptions: BrandingType[] = [ 'hpe', 'determined' ];

const knobTypeOptions = enumToOptions<Orientation>(Orientation);

export const Default = (): React.ReactNode => (
  <Logo branding="determined" orientation={Orientation.Horizontal} />
);

export const Custom = (): React.ReactNode => (
  <Logo
    branding={select<BrandingType>('Branding', knobBrandingOptions, 'determined')}
    orientation={select<Orientation>('Orientation', knobTypeOptions, Orientation.Horizontal)}
  />
);
