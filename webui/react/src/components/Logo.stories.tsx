import { select, withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import { enumToOptions } from 'storybook/utils';
import { BrandingType } from 'types';

import Logo, { LogoType } from './Logo';

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

const knobBrandingOptions = enumToOptions<BrandingType>(BrandingType);
const knobTypeOptions = enumToOptions<LogoType>(BrandingType);

export const Default = (): React.ReactNode => (
  <Logo branding={BrandingType.Determined} type={LogoType.OnLightHorizontal} />
);

export const Custom = (): React.ReactNode => (
  <Logo
    branding={select<BrandingType>('Branding', knobBrandingOptions, BrandingType.Determined)}
    type={select<LogoType>('Type', knobTypeOptions, LogoType.OnLightHorizontal)}
  />
);
