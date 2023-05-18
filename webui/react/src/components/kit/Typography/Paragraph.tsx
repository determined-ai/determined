import React from 'react';

import { ValueOf } from 'shared/types';
import { ThemeFont } from 'utils/fonts';

import css from './index.module.scss';

export const ParagraphSizes = {
  default: 'default',
  L: 'L',
  S: 'S',
  XL: 'XL',
  XS: 'XS',
} as const;

export type ParagraphSizes = ValueOf<typeof ParagraphSizes>;
interface Props {
  size?: ParagraphSizes;
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

    if (size === ParagraphSizes.XL) return css[`${lineType}LineXL`];
    if (size === ParagraphSizes.L) return css[`${lineType}LineL`];
    if (size === ParagraphSizes.default) return css[`${lineType}LineDefault`];
    if (size === ParagraphSizes.S) return css[`${lineType}LineS`];

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
