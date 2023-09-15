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

/** CIELAB color space */
interface CIELAB {
  a: number;
  b: number;
  /** perceptual lightness */
  l: number;
}

const hexRegex = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i;
const hslRegex = /^hsl\(\d+,\s*\d+%,\s*\d+%\)$/i;
const rgbaRegex = /^rgba?\(\s*?(\d+)\s*?,\s*?(\d+)\s*?,\s*?(\d+)\s*?(,\s*?([\d.]+)\s*?)?\)$/i;

export const isColor = (color: string): boolean => {
  return hexRegex.test(color) || hslRegex.test(color) || rgbaRegex.test(color);
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

export const rgba2str = (rgba: RgbaColor): string => {
  if (rgba.a != null) {
    return `rgba(${rgba.r}, ${rgba.g}, ${rgba.b}, ${rgba.a})`;
  }
  return `rgb(${rgba.r}, ${rgba.g}, ${rgba.b})`;
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
  const [adr, adg, adb, ada] = [dr, dg, db, da].map((x) => Math.abs(x));
  const delta = adr + adg + adb + 255 * ada;
  if (delta === 0) return rgba0;

  const [pr, pg, pb, pa] = [dr, dg, db, da].map((x) => (x * amount) / delta);
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

/** convert rgb to CIELAB color. ignores alpha values */
export const rgb2lab = (rgb: RgbaColor): CIELAB => {
  const [r, g, b] = [rgb.r / 255, rgb.g / 255, rgb.b / 255];
  const [x, y, z] = [
    r * 0.4124 + g * 0.3576 + b * 0.1805,
    r * 0.2126 + g * 0.7152 + b * 0.0722,
    r * 0.0193 + g * 0.1192 + b * 0.9505,
  ];
  const [l, a, bb] = [116 * y ** 2 - 16, 500 * (x - y), 200 * (y - z)];
  return { a, b: bb, l };
};

/** calculate euclidean distance between two n dimentional points */
const pointDistance = (p0: number[], p1: number[]): number => {
  let sum = 0;
  if (p0.length !== p1.length) {
    throw new Error('points must be of same dimension');
  }
  for (let i = 0; i < p0.length; i++) {
    sum += (p1[i] - p0[i]) ** 2;
  }
  return Math.sqrt(sum);
};

/** calculate euclidean distance between two CIELAB colors */
export const labDistance = (lab0: CIELAB, lab1: CIELAB): number => {
  return pointDistance([lab0.l, lab0.a, lab0.b], [lab1.l, lab1.a, lab1.b]);
};
