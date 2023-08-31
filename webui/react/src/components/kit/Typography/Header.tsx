import React from 'react';

import { TypographySize } from 'components/kit/internal/fonts';
import css from 'components/kit/Typography/index.module.scss';
interface Props {
  size?: TypographySize;
}

const Header: React.FC<React.PropsWithChildren<Props>> = ({ children, size }) => {
  const getThemeClass = () => {
    if (!size) return '';

    if (size === TypographySize.XL) return css.headerXL;
    if (size === TypographySize.L) return css.headerL;
    if (size === TypographySize.default) return css.headerDefault;
    if (size === TypographySize.S) return css.headerS;

    return css.headerXS;
  };

  const classes = [css.header, getThemeClass()];

  return <h1 className={classes.join(' ')}>{children}</h1>;
};

export default Header;
