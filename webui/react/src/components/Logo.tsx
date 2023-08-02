import React, { useMemo } from 'react';

import logoDeterminedOnDarkHorizontal from 'assets/images/logo-determined-on-dark-horizontal.svg?url';
import logoDeterminedOnDarkVertical from 'assets/images/logo-determined-on-dark-vertical.svg?url';
import logoDeterminedOnLightHorizontal from 'assets/images/logo-determined-on-light-horizontal.svg?url';
import logoDeterminedOnLightVertical from 'assets/images/logo-determined-on-light-vertical.svg?url';
import logoHpeOnDarkHorizontal from 'assets/images/logo-hpe-on-dark-horizontal.svg?url';
import logoHpeOnLightHorizontal from 'assets/images/logo-hpe-on-light-horizontal.svg?url';
import { serverAddress } from 'routes/utils';
import useUI from 'stores/contexts/UI';
import { BrandingType } from 'stores/determinedInfo';
import { ValueOf } from 'types';
import { reactHostAddress } from 'utils/routes';
import { DarkLight } from 'utils/themes';

import css from './Logo.module.scss';

export const Orientation = {
  Horizontal: 'horizontal',
  Vertical: 'vertical',
} as const;

export type Orientation = ValueOf<typeof Orientation>;

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
  const { ui } = useUI();
  const classes = [css[branding], css[orientation]];

  const alt = useMemo(() => {
    const isDetermined = branding === BrandingType.Determined;
    const server = serverAddress();
    const isSameServer = reactHostAddress() === server;
    return [
      isDetermined ? 'Determined AI Logo' : 'HPE Machine Learning Development Logo',
      isSameServer ? '' : ` (Server: ${server})`,
    ].join();
  }, [branding]);

  return (
    <img alt={alt} className={classes.join(' ')} src={logos[branding][orientation][ui.darkLight]} />
  );
};

export default Logo;
