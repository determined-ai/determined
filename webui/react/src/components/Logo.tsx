import React from 'react';

import logoOnDarkHorizontal from 'assets/logo-on-dark-horizontal.svg';
import logoOnDarkVertical from 'assets/logo-on-dark-vertical.svg';
import logoOnLightHorizontal from 'assets/logo-on-light-horizontal.svg';
import logoOnLightVertical from 'assets/logo-on-light-vertical.svg';
import { reactHostAddress, serverAddress } from 'routes/utils';
import { PropsWithClassName } from 'types';

import css from './Logo.module.scss';

export enum LogoTypes {
  OnDarkHorizontal = 'on-dark-horizontal',
  OnDarkVertical = 'on-dark-vertical',
  OnLightHorizontal = 'on-light-horizontal',
  OnLightVertical = 'on-light-vertical',
}

interface Props {
  type: LogoTypes;
}

const logos: Record<LogoTypes, string> = {
  [LogoTypes.OnDarkHorizontal]: logoOnDarkHorizontal,
  [LogoTypes.OnDarkVertical]: logoOnDarkVertical,
  [LogoTypes.OnLightHorizontal]: logoOnLightHorizontal,
  [LogoTypes.OnLightVertical]: logoOnLightVertical,
};

const Logo: React.FC<PropsWithClassName<Props>> = (props: PropsWithClassName<Props>) => {
  let alt = 'Determined AI Logo';
  if (reactHostAddress() !== serverAddress()) alt += ` (Server: ${serverAddress()})`;
  return (
    <img
      alt={alt}
      className={`${css.base} ${css[props.type]}`}
      src={logos[props.type]} />
  );
};

export default Logo;
