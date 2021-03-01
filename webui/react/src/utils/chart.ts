import themes, { defaultThemeId } from 'themes';
import { Primitive, Range } from 'types';
import { primitiveSorter } from 'utils/sort';

import { ColorScale } from './color';
import { clone } from './data';

/* Color Scales */

const DEFAULT_SCALE_COLORS: Range<string> = [
  themes[defaultThemeId].colors.danger.light,
  themes[defaultThemeId].colors.action.normal,
];
const REVERSE_SCALE_COLORS = clone(DEFAULT_SCALE_COLORS).reverse();
const NEUTRAL_SCALE_COLORS: Range<string> = [
  'rgb(255, 184, 0)',
  themes[defaultThemeId].colors.action.normal,
];

export const getColorScale = (range?: Range<number>, smallerIsBetter?: boolean): ColorScale[] => {
  let colors = NEUTRAL_SCALE_COLORS;
  if (smallerIsBetter != null) {
    colors = smallerIsBetter ? REVERSE_SCALE_COLORS : DEFAULT_SCALE_COLORS;
  }
  return colors.map((color, index): ColorScale => {
    if (range) {
      const scale = range ? range[index] : index;
      return { color, scale };
    }
    return { color, scale: index };
  });
};

/* Ranges */

export const defaultNumericRange = (reverse = false): Range<number> => {
  const range: Range<number> = [ Number.NEGATIVE_INFINITY, Number.POSITIVE_INFINITY ];
  if (reverse) range.reverse();
  return range;
};

export const getNumericRange = (values: number[], forceRange = true): Range<number> | undefined => {
  if (values.length === 0) return;
  const range = values.reduce((acc, value) => {
    acc[0] = Math.min(acc[0], value);
    acc[1] = Math.max(acc[1], value);
    return acc;
  }, [ Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY ] as Range<number>);

  if (forceRange && range[0] === range[1]) {
    range[0] = Math.floor(range[0]);
    range[1] = Math.ceil(range[1]);
  }

  return range;
};

export const updateRange = <T extends Primitive>(
  range: Range<T> | undefined,
  value: T,
): Range<T> => {
  if (!range) return [ value, value ];
  return [
    primitiveSorter(range[0], value) === -1 ? range[0] : value,
    primitiveSorter(range[1], value) === 1 ? range[1] : value,
  ];
};

export const normalizeRange = (values: number[], range: Range<number>): number[] => {
  if (range[1] === range[0]) return values;

  const diff = range[1] - range[0];
  return values.map(value => (value - range[0]) / diff);
};

export function distance(x0: number, y0: number, x1: number, y1: number): number {
  return Math.sqrt((x1 - x0) ** 2 + (y1 - y0) ** 2);
}
