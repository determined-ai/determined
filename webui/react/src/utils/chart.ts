import { Primitive, Range } from 'shared/types';
import { ColorScale } from 'shared/utils/color';
import { clone } from 'shared/utils/data';
import { Theme } from 'themes';
import { primitiveSorter } from 'utils/sort';

/* Color Scales */

export const getColorScale = (
  theme: Theme,
  range?: Range<number>,
  smallerIsBetter?: boolean,
): ColorScale[] => {
  const defaultScale = [ theme.statusCriticalWeak, theme.statusActive ];
  const reverseScale = clone(defaultScale).reverse();
  let colors = [ theme.statusWarningStrong, theme.statusActive ];

  if (smallerIsBetter != null) {
    colors = smallerIsBetter ? reverseScale : defaultScale;
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
