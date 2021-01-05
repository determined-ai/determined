import { GLASBEY } from 'constants/colors';

interface HslColor {
  h: number;
  l: number;
  s: number;
}

export const hex2hsl = (hex: string): HslColor => {
  const hsl: HslColor = { h: Math.round(Math.random() * 6), l: 0.5, s: 0.5 };
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);

  if (result && result.length > 3) {
    const r = parseInt(result[1], 16) / 255;
    const g = parseInt(result[2], 16) / 255;
    const b = parseInt(result[3], 16) / 255;
    const max = Math.max(r, g, b);
    const min = Math.min(r, g, b);
    const avg = (max + min) / 2;

    hsl.h = hsl.s = hsl.l = avg;

    if (max === min) {
      hsl.h = hsl.s = 0; // achromatic
    } else {
      const d = max - min;
      hsl.s = hsl.l > 0.5 ? d / (2 - max - min) : d / (max + min);
      switch (max) {
        case r: hsl.h = (g - b) / d + (g < b ? 6 : 0); break;
        case g: hsl.h = (b - r) / d + 2; break;
        case b: hsl.h = (r - g) / d + 4; break;
      }
    }
  }

  hsl.h = Math.round(360 * hsl.h / 6);
  hsl.s = Math.round(hsl.s * 100);
  hsl.l = Math.round(hsl.l * 100);

  return hsl;
};

export const hsl2str = (hsl: HslColor): string => {
  return `hsl(${hsl.h}, ${hsl.s}%, ${hsl.l}%)`;
};

export const glasbeyColor = (seriesIdx: number): string => {
  const rgb = GLASBEY[seriesIdx];
  return `rgb(${rgb[0]}, ${rgb[1]}, ${rgb[2]})`;
};
