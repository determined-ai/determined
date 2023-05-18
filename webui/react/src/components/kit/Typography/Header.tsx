import React from 'react';

import { TypographySizes } from 'utils/fonts';

import css from './index.module.scss';
interface Props {
  size?: TypographySizes;
}

const Header: React.FC<React.PropsWithChildren<Props>> = ({ children, size }) => {
  const getThemeClass = () => {
    if (!size) return '';

    if (size === TypographySizes.XL) return css.headerXL;
    if (size === TypographySizes.L) return css.headerL;
    if (size === TypographySizes.default) return css.headerDefault;
    if (size === TypographySizes.S) return css.headerS;

    return css.headerXS;
  };

  const classes = [css.header, getThemeClass()];

  return <h1 className={classes.join(' ')}>{children}</h1>;
};

export default Header;
