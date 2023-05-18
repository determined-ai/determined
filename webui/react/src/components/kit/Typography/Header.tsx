import React from 'react';

import { ValueOf } from 'shared/types';

import css from './index.module.scss';

export const HeaderSizes = {
  default: 'default',
  L: 'L',
  S: 'S',
  XL: 'XL',
  XS: 'XS',
} as const;

export type HeaderSizes = ValueOf<typeof HeaderSizes>;
interface Props {
  size?: HeaderSizes;
}

const Header: React.FC<React.PropsWithChildren<Props>> = ({ children, size }) => {
  const getThemeClass = () => {
    if (!size) return '';

    if (size === HeaderSizes.XL) return css.headerXL;
    if (size === HeaderSizes.L) return css.headerL;
    if (size === HeaderSizes.default) return css.headerDefault;
    if (size === HeaderSizes.S) return css.headerS;

    return css.headerXS;
  };

  const classes = [css.header, getThemeClass()];

  return <h1 className={classes.join(' ')}>{children}</h1>;
};

export default Header;
