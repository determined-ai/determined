import { GLASBEY } from 'constants/colors';

import * as util from './color';

describe('color utility', () => {
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

  describe('glasbeyColor', () => {
    const firstColor = 'rgb(87, 43, 255)';
    const lastColor = 'rgb(142, 190, 88)';

    it('should match first glasbey color', () => {
      expect(util.glasbeyColor(0)).toBe(firstColor);
    });

    it('should match last glasbey color', () => {
      expect(util.glasbeyColor(GLASBEY.length - 1)).toBe(lastColor);
    });

    it('should wrap around the list of glasbey colors', () => {
      expect(util.glasbeyColor(GLASBEY.length)).toBe(firstColor);
    });
  });

  describe('hex2hsl', () => {
    it('should convert all hex colors to hsl', () => {
      colors.forEach(color => {
        expect(util.hex2hsl(color.hex)).toEqual(color.hsl);
      });
    });
  });

  describe('hex2rgb', () => {
    it('should convert all hex colors to rgb', () => {
      colors.forEach(color => {
        expect(util.hex2rgb(color.hex)).toEqual(color.rgb);
      });
    });
  });

  describe('hsl2str', () => {
    it('should convert all hsl colors to hsl string', () => {
      colors.forEach(color => {
        expect(util.hsl2str(color.hsl)).toEqual(color.hslStr);
      });
    });
  });

  describe('rgba2str', () => {
    it('should convert all rgba colors to rgba string', () => {
      colors.forEach(color => {
        expect(util.rgba2str(color.rgb)).toEqual(color.rgbStr);
      });
    });
  });

  describe('rgbaFromGradient', () => {
    it('should interpolate grey', () => {
      const black = { b: 0, g: 0, r: 0 };
      const white = { b: 255, g: 255, r: 255 };
      const grey = { b: 128, g: 128, r: 128 };
      expect(util.rgbaFromGradient(black, white, 0.5)).toEqual(grey);
      expect(util.rgbaFromGradient(white, black, 0.5)).toEqual(grey);
    });

    it('should interpolate alpha', () => {
      const black = { a: 1.0, b: 0, g: 0, r: 0 };
      const white = { a: 0.0, b: 255, g: 255, r: 255 };
      const grey = { a: 0.5, b: 128, g: 128, r: 128 };
      expect(util.rgbaFromGradient(black, white, 0.5)).toEqual(grey);
      expect(util.rgbaFromGradient(white, black, 0.5)).toEqual(grey);
    });
  });

  describe('str2rgba', () => {
    it('should convert all hex colors to rgba', () => {
      colors.forEach(color => {
        expect(util.str2rgba(color.hex)).toEqual(color.rgb);
      });
    });

    it('should convert all rgba string colors to rgba', () => {
      expect(util.str2rgba('rgba(255, 128, 64, 0.5)')).toEqual({ a: 0.5, b: 64, g: 128, r: 255 });
    });
  });
});
