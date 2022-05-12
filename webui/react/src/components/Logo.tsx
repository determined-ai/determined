import React, { useMemo } from 'react';

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
import { BrandingType } from 'types';

import { reactHostAddress } from '../shared/utils/routes';

import css from './Logo.module.scss';

export enum LogoType {
  OnDarkHorizontal = 'onDarkHorizontal',
  OnDarkVertical = 'onDarkVertical',
  OnLightHorizontal = 'onLightHorizontal',
  OnLightVertical = 'onLightVertical',
}

interface Props {
  branding: BrandingType;
  type: LogoType;
}

const logos: Record<BrandingType, Record<LogoType, string>> = {
  [BrandingType.Determined]: {
    [LogoType.OnDarkHorizontal]: logoDeterminedOnDarkHorizontal,
    [LogoType.OnDarkVertical]: logoDeterminedOnDarkVertical,
    [LogoType.OnLightHorizontal]: logoDeterminedOnLightHorizontal,
    [LogoType.OnLightVertical]: logoDeterminedOnLightVertical,
  },
  [BrandingType.HPE]: {
    [LogoType.OnDarkHorizontal]: logoHpeOnDarkHorizontal,
    [LogoType.OnDarkVertical]: logoHpeOnDarkHorizontal,
    [LogoType.OnLightHorizontal]: logoHpeOnLightHorizontal,
    [LogoType.OnLightVertical]: logoHpeOnLightHorizontal,
  },
};

const Logo: React.FC<Props> = ({ branding, type }: Props) => {
  const classes = [ css[branding], css[type] ];

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
      src={logos[branding][type]}
    />
  );
};

export default Logo;
