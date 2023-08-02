import { ValueOf } from 'components/kit/internal/types';

export const ThemeFont = {
  Code: 'var(--theme-font-family-code)',
  UI: 'var(--theme-font-family)',
} as const;

export const TypographySize = {
  default: 'default',
  L: 'L',
  S: 'S',
  XL: 'XL',
  XS: 'XS',
} as const;

export type TypographySize = ValueOf<typeof TypographySize>;
