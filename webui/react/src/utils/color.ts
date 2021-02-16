import { GLASBEY } from 'constants/colors';

/*
 * h - hue between 0 and 360
 * s - saturation between 0.0 and 1.0
 * l - lightness between 0.0 and 1.0
 */
interface HslColor {
  h: number;
  l: number;
  s: number;
}

/*
 * r - red between 0 and 255
 * g - green between 0 and 255
 * b - blue between 0 and 255
 * a - alpha between 0.0 and 1.0
 */
interface RgbaColor {
  a?: number;
  b: number;
  g: number;
  r: number;
}

export interface ColorScale {
  color: string;    // rgb(a) or hex color
  scale: number;    // scale between 0.0 and 1.0
}

export const glasbeyColor = (seriesIdx: number): string => {
  const index = seriesIdx % GLASBEY.length;
  const rgb = GLASBEY[index];
  return `rgb(${rgb[0]}, ${rgb[1]}, ${rgb[2]})`;
};

export const hex2hsl = (hex: string): HslColor => {
  const rgb = hex2rgb(hex);
  const r = rgb.r / 255;
  const g = rgb.g / 255;
  const b = rgb.b / 255;
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  const avg = (max + min) / 2;
  const hsl: HslColor = { h: Math.round(Math.random() * 6), l: 0.5, s: 0.5 };

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

  hsl.h = Math.round(360 * hsl.h / 6);
  hsl.s = Math.round(hsl.s * 100);
  hsl.l = Math.round(hsl.l * 100);

  return hsl;
};

export const hex2rgb = (hex: string): RgbaColor => {
  const rgba = { b: 0, g: 0, r: 0 };
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);

  if (result && result.length > 3) {
    rgba.r = parseInt(result[1], 16);
    rgba.g = parseInt(result[2], 16);
    rgba.b = parseInt(result[3], 16);
  }

  return rgba;
};

export const hsl2str = (hsl: HslColor): string => {
  return `hsl(${hsl.h}, ${hsl.s}%, ${hsl.l}%)`;
};

export const rgba2str = (rgba: RgbaColor): string => {
  if (rgba.a != null) {
    return `rgba(${rgba.r}, ${rgba.g}, ${rgba.b}, ${rgba.a})`;
  }
  return `rgb(${rgba.r}, ${rgba.g}, ${rgba.b})`;
};

export const rgbaFromGradient = (
  rgba0: RgbaColor,
  rgba1: RgbaColor,
  distance: number,
): RgbaColor => {
  const r = Math.round((rgba1.r - rgba0.r) * distance + rgba0.r);
  const g = Math.round((rgba1.g - rgba0.g) * distance + rgba0.g);
  const b = Math.round((rgba1.b - rgba0.b) * distance + rgba0.b);

  if (rgba0.a != null && rgba1.a != null) {
    const a = (rgba1.a - rgba0.a) * distance + rgba0.a;
    return { a, b, g, r };
  }

  return { b, g, r };
};

export const str2rgba = (str: string): RgbaColor => {
  if (/^#/.test(str)) return hex2rgb(str);

  const regex = /^rgba?\(\s*?(\d+)\s*?,\s*?(\d+)\s*?,\s*?(\d+)\s*?(,\s*?([\d.]+)\s*?)?\)$/i;
  const result = regex.exec(str);
  if (result && result.length > 3) {
    const rgba = { a: 1.0, b: 0, g: 0, r: 0 };
    rgba.r = parseInt(result[1]);
    rgba.g = parseInt(result[2]);
    rgba.b = parseInt(result[3]);
    if (result.length > 5) rgba.a = parseInt(result[5]);
    return rgba;
  }

  return { a: 0.0, b: 0, g: 0, r: 0 };
};
