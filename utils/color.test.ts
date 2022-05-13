import { GLASBEY } from 'shared/constants/colors';

import * as utils from './color';

describe('Color Utilities', () => {
  const colors = [
    {
      hex: '#000000',
      hsl: { h: 0, l: 0, s: 0 },
      hslStr: 'hsl(0, 0%, 0%)',
      rgb: { b: 0, g: 0, r: 0 },
      rgbStr: 'rgb(0, 0, 0)',
    },
    {
      hex: '#999999',
      hsl: { h: 0, l: 60, s: 0 },
      hslStr: 'hsl(0, 0%, 60%)',
      rgb: { b: 153, g: 153, r: 153 },
      rgbStr: 'rgb(153, 153, 153)',
    },
    {
      hex: '#ffffff',
      hsl: { h: 0, l: 100, s: 0 },
      hslStr: 'hsl(0, 0%, 100%)',
      rgb: { b: 255, g: 255, r: 255 },
      rgbStr: 'rgb(255, 255, 255)',
    },
    {
      hex: '#ff0000',
      hsl: { h: 0, l: 50, s: 100 },
      hslStr: 'hsl(0, 100%, 50%)',
      rgb: { b: 0, g: 0, r: 255 },
      rgbStr: 'rgb(255, 0, 0)',
    },
    {
      hex: '#00ff00',
      hsl: { h: 120, l: 50, s: 100 },
      hslStr: 'hsl(120, 100%, 50%)',
      rgb: { b: 0, g: 255, r: 0 },
      rgbStr: 'rgb(0, 255, 0)',
    },
    {
      hex: '#0000ff',
      hsl: { h: 240, l: 50, s: 100 },
      hslStr: 'hsl(240, 100%, 50%)',
      rgb: { b: 255, g: 0, r: 0 },
      rgbStr: 'rgb(0, 0, 255)',
    },
  ];

  describe('isColor', () => {
    it('should match hex color', () => {
      colors.forEach(color => {
        expect(utils.isColor(color.hex)).toEqual(true);
      });
    });

    it('should match hsl color', () => {
      colors.forEach(color => {
        expect(utils.isColor(color.hslStr)).toEqual(true);
      });
    });

    it('should match rgba color', () => {
      colors.forEach(color => {
        expect(utils.isColor(color.rgbStr)).toEqual(true);
      });
    });

    it('should not match invalid colors', () => {
      expect(utils.isColor('')).toEqual(false);
      expect(utils.isColor('abc')).toEqual(false);
      expect(utils.isColor('#12345678')).toEqual(false);
      expect(utils.isColor('rgba()')).toEqual(false);
      expect(utils.isColor('hsl()')).toEqual(false);
    });
  });

  describe('glasbeyColor', () => {
    const firstColor = 'rgb(0, 155, 222)';
    const lastColor = 'rgb(142, 190, 88)';

    it('should match first glasbey color', () => {
      expect(utils.glasbeyColor(0)).toBe(firstColor);
    });

    it('should match last glasbey color', () => {
      expect(utils.glasbeyColor(GLASBEY.length - 1)).toBe(lastColor);
    });

    it('should wrap around the list of glasbey colors', () => {
      expect(utils.glasbeyColor(GLASBEY.length)).toBe(firstColor);
    });
  });

  describe('hex2hsl', () => {
    it('should convert all hex colors to hsl', () => {
      colors.forEach(color => {
        expect(utils.hex2hsl(color.hex)).toEqual(color.hsl);
      });
    });
  });

  describe('hex2rgb', () => {
    it('should convert all hex colors to rgb', () => {
      colors.forEach(color => {
        expect(utils.hex2rgb(color.hex)).toEqual(color.rgb);
      });
    });
  });

  describe('hsl2str', () => {
    it('should convert all hsl colors to hsl string', () => {
      colors.forEach(color => {
        expect(utils.hsl2str(color.hsl)).toEqual(color.hslStr);
      });
    });
  });

  describe('rgba2str', () => {
    it('should convert all rgb colors to rgb string', () => {
      colors.forEach(color => {
        expect(utils.rgba2str(color.rgb)).toEqual(color.rgbStr);
      });
    });

    it('should convert all rgba colors to rgba string', () => {
      expect(utils.rgba2str({ a: 0.5, b: 50, g: 50, r: 50 })).toBe('rgba(50, 50, 50, 0.5)');
      expect(utils.rgba2str({ a: 1, b: 0, g: 128, r: 255 })).toBe('rgba(255, 128, 0, 1)');
    });
  });

  describe('rgbaFromGradient', () => {
    it('should interpolate grey', () => {
      const black = { b: 0, g: 0, r: 0 };
      const white = { b: 255, g: 255, r: 255 };
      const grey = { b: 128, g: 128, r: 128 };
      expect(utils.rgbaFromGradient(black, white, 0.5)).toEqual(grey);
      expect(utils.rgbaFromGradient(white, black, 0.5)).toEqual(grey);
    });

    it('should interpolate alpha', () => {
      const black = { a: 1.0, b: 0, g: 0, r: 0 };
      const white = { a: 0.0, b: 255, g: 255, r: 255 };
      const grey = { a: 0.5, b: 128, g: 128, r: 128 };
      expect(utils.rgbaFromGradient(black, white, 0.5)).toEqual(grey);
      expect(utils.rgbaFromGradient(white, black, 0.5)).toEqual(grey);
    });
  });

  describe('rgbaMix', () => {
    const black = { b: 0, g: 0, r: 0 };
    const white = { b: 255, g: 255, r: 255 };
    const color = { b: 200, g: 150, r: 100 };
    const amount = 33;

    it('should mix rgba colors', () => {
      const result0 = { a: 1.0, b: 185.33333333333334, g: 139, r: 92.66666666666667 };
      const result1 = { a: 1.0, b: 205.76190476190476, g: 161, r: 116.23809523809524 };
      expect(utils.rgbaMix(color, black, amount, false)).toStrictEqual(result0);
      expect(utils.rgbaMix(color, white, amount, false)).toStrictEqual(result1);
    });

    it('should round mixed results', () => {
      const result0 = { a: 1.0, b: 185, g: 139, r: 93 };
      const result1 = { a: 1.0, b: 206, g: 161, r: 116 };
      expect(utils.rgbaMix(color, black, amount)).toStrictEqual(result0);
      expect(utils.rgbaMix(color, white, amount)).toStrictEqual(result1);
    });
  });

  describe('str2rgba', () => {
    it('should convert all hex colors to rgba', () => {
      colors.forEach(color => {
        expect(utils.str2rgba(color.hex)).toEqual(color.rgb);
      });
    });

    it('should convert all rgba string colors to rgba', () => {
      expect(utils.str2rgba('rgba(255, 128, 64, 0.5)')).toEqual({ a: 0.5, b: 64, g: 128, r: 255 });
    });

    it('should handle invalid rgba string colors', () => {
      expect(utils.str2rgba('rgba(1000, 1000, 1000, 10')).toEqual({ a: 0.0, b: 0, g: 0, r: 0 });
    });
  });
});
