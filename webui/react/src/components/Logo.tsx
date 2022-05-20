import React, { useMemo } from 'react';

import useTheme from 'hooks/useTheme';
import { serverAddress } from 'routes/utils';
import logoDeterminedOnDarkHorizontal from
  'shared/assets/images/logo-determined-on-dark-horizontal.svg';
import logoDeterminedOnDarkVertical from
  'shared/assets/images/logo-determined-on-dark-vertical.svg';
import logoDeterminedOnLightHorizontal from
  'shared/assets/images/logo-determined-on-light-horizontal.svg';
import logoDeterminedOnLightVertical from
  'shared/assets/images/logo-determined-on-light-vertical.svg';
import logoHpeOnDarkHorizontal from 'shared/assets/images/logo-hpe-on-dark-horizontal.svg';
import logoHpeOnLightHorizontal from 'shared/assets/images/logo-hpe-on-light-horizontal.svg';
import { DarkLight } from 'themes';
import { BrandingType } from 'types';

import { reactHostAddress } from '../shared/utils/routes';

import css from './Logo.module.scss';

export enum Orientation {
  Horizontal = 'horizontal',
  Vertical = 'vertical',
}

interface Props {
  branding: BrandingType;
  orientation: Orientation;
}

const logos: Record<BrandingType, Record<Orientation, Record<DarkLight, string>>> = {
  [BrandingType.Determined]: {
    [Orientation.Horizontal]: {
      [DarkLight.Dark]: logoDeterminedOnDarkHorizontal,
      [DarkLight.Light]: logoDeterminedOnLightHorizontal,
    },
    [Orientation.Vertical]: {
      [DarkLight.Dark]: logoDeterminedOnDarkVertical,
      [DarkLight.Light]: logoDeterminedOnLightVertical,
    },
  },
  [BrandingType.HPE]: {
    [Orientation.Horizontal]: {
      [DarkLight.Dark]: logoHpeOnDarkHorizontal,
      [DarkLight.Light]: logoHpeOnLightHorizontal,
    },
    [Orientation.Vertical]: {
      [DarkLight.Dark]: logoHpeOnDarkHorizontal,
      [DarkLight.Light]: logoHpeOnLightHorizontal,
    },
  },
};

const Logo: React.FC<Props> = ({ branding, orientation }: Props) => {
  const { mode } = useTheme();
  const classes = [ css[branding], css[orientation] ];

  const alt = useMemo(() => {
    const isDetermined = branding === BrandingType.Determined;
    const server = serverAddress();
    const isSameServer = reactHostAddress() === server;
    return [
      isDetermined ? 'Determined AI Logo' : 'HPE Machine Learning Development Logo',
      isSameServer ? '' : ` (Server: ${server})`,
    ].join();
  }, [ branding ]);

  return (
    <img
      alt={alt}
      className={classes.join(' ')}
      src={logos[branding][orientation][mode]}
    />
  );
};

export default Logo;
