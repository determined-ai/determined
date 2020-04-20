import React from 'react';

import logoLight from 'assets/logo-on-dark-horizontal.svg';
import logoDark from 'assets/logo-on-light-horizontal.svg';
import { PropsWithClassName } from 'types';

import css from './Logo.module.scss';

export enum LogoTypes {
  Light,
  Dark,
}

interface Props {
  type: LogoTypes;
}

const logos: Record<LogoTypes, string> = {
  [LogoTypes.Light]: logoLight,
  [LogoTypes.Dark]: logoDark,
};

const Logo: React.FC<PropsWithClassName<Props>> = (props: PropsWithClassName<Props>) => {
  return <img alt="Determined AI Logo" className={`${css.base} ${props.className}`}
    src={logos[props.type]} />;
};

export default Logo;
