import { GLASBEY } from 'shared/constants/colors';

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

const hexRegex = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i;
const hslRegex = /^hsl\(\d+,\s*\d+%,\s*\d+%\)$/i;
const rgbaRegex = /^rgba?\(\s*?(\d+)\s*?,\s*?(\d+)\s*?,\s*?(\d+)\s*?(,\s*?([\d.]+)\s*?)?\)$/i;

export const isColor = (color: string): boolean => {
  return hexRegex.test(color) || hslRegex.test(color) || rgbaRegex.test(color);
};

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
  const rgb = { b: 0, g: 0, r: 0 };
  const result = hexRegex.exec(hex);

  if (result && result.length > 3) {
    rgb.r = parseInt(result[1], 16);
    rgb.g = parseInt(result[2], 16);
    rgb.b = parseInt(result[3], 16);
  }

  return rgb;
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
  percent: number,
): RgbaColor => {
  const r = Math.round((rgba1.r - rgba0.r) * percent + rgba0.r);
  const g = Math.round((rgba1.g - rgba0.g) * percent + rgba0.g);
  const b = Math.round((rgba1.b - rgba0.b) * percent + rgba0.b);

  if (rgba0.a != null && rgba1.a != null) {
    const a = (rgba1.a - rgba0.a) * percent + rgba0.a;
    return { a, b, g, r };
  }

  return { b, g, r };
};

export const rgbaMix = (
  rgba0: RgbaColor,
  rgba1: RgbaColor,
  amount: number,
  rounded = true,
): RgbaColor => {
  const dr = rgba1.r - rgba0.r;
  const dg = rgba1.g - rgba0.g;
  const db = rgba1.b - rgba0.b;
  const da = (rgba1.a ?? 1.0) - (rgba0.a ?? 1.0);
  const [ adr, adg, adb, ada ] = [ dr, dg, db, da ].map(x => Math.abs(x));
  const delta = adr + adg + adb + (255 * ada);
  if (delta === 0) return rgba0;

  const [ pr, pg, pb, pa ] = [ dr, dg, db, da ].map(x => x * amount / delta);
  const r = Math.min(255, Math.max(0, rgba0.r + pr));
  const g = Math.min(255, Math.max(0, rgba0.g + pg));
  const b = Math.min(255, Math.max(0, rgba0.b + pb));
  const a = Math.min(1.0, Math.max(0.0, (rgba0.a ?? 1.0) + pa));
  return {
    a,
    b: rounded ? Math.round(b) : b,
    g: rounded ? Math.round(g) : g,
    r: rounded ? Math.round(r) : r,
  };
};

export const str2rgba = (str: string): RgbaColor => {
  if (hexRegex.test(str)) return hex2rgb(str);

  const regex = rgbaRegex;
  const result = regex.exec(str);
  if (result && result.length > 3) {
    const rgba = { a: 1.0, b: 0, g: 0, r: 0 };
    rgba.r = parseInt(result[1]);
    rgba.g = parseInt(result[2]);
    rgba.b = parseInt(result[3]);
    if (result.length > 5 && result[5] != null) rgba.a = parseFloat(result[5]);
    return rgba;
  }

  return { a: 0.0, b: 0, g: 0, r: 0 };
};
