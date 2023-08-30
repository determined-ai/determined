import React from 'react';

import { ThemeFont, TypographySize } from 'components/kit/internal/fonts';
import css from 'components/kit/Typography/index.module.scss';

interface Props {
  size?: TypographySize;
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

    if (size === TypographySize.XL) return css[`${lineType}LineXL`];
    if (size === TypographySize.L) return css[`${lineType}LineL`];
    if (size === TypographySize.default) return css[`${lineType}LineDefault`];
    if (size === TypographySize.S) return css[`${lineType}LineS`];

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
