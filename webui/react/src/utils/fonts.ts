import { ValueOf } from 'shared/types';

export const ThemeFont = {
  Code: 'var(--theme-font-family-code)',
  UI: 'var(--theme-font-family)',
} as const;

export const TypographySizes = {
  default: 'default',
  L: 'L',
  S: 'S',
  XL: 'XL',
  XS: 'XS',
} as const;

export type TypographySizes = ValueOf<typeof TypographySizes>;
