import React from 'react';

import Logo, { LogoTypes } from './Logo';

export default {
  component: Logo,
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

export const Default = (): React.ReactNode => <Logo type={LogoTypes.OnLightHorizontal} />;
