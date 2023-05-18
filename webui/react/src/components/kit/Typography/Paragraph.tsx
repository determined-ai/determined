import React from 'react';

import { ThemeFont, TypographySizes } from 'utils/fonts';

import css from './index.module.scss';

interface Props {
  size?: TypographySizes;
  type?: 'single line' | 'multi line';
  font?: 'code' | 'ui';
}

const Paragraph: React.FC<React.PropsWithChildren<Props>> = ({
  children,
  font = 'ui',
  size,
  type = 'single line',
}) => {
  const getThemeClass = () => {
    if (!size) return '';

    const lineType = type.replace(' line', '');

    if (size === TypographySizes.XL) return css[`${lineType}LineXL`];
    if (size === TypographySizes.L) return css[`${lineType}LineL`];
    if (size === TypographySizes.default) return css[`${lineType}LineDefault`];
    if (size === TypographySizes.S) return css[`${lineType}LineS`];

    return css[`${lineType}LineXS`];
  };

  return (
    <p
      className={getThemeClass()}
      style={{ fontFamily: `${ThemeFont[font === 'code' ? 'Code' : 'UI']}` }}>
      {children}
    </p>
  );
};

export default Paragraph;
