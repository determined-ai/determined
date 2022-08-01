import * as themes from './themes';
import { isColor, maxColorDistance, str2rgba } from './utils/color';

const supportedThemes = {
  darkDet: themes.themeDarkDetermined,
  darkHpe: themes.themeDarkHpe,
  lightDet: themes.themeLightDetermined,
  lightHpe: themes.themeLightHpe,
};

/** pars of theme colors used next to eachother */
const pairs: (keyof themes.Theme)[][] = [
  [ 'ixActive', 'ixOnActive' ],
  [ 'ixActive', 'ixBorderActive' ],
  [ 'ixInactive', 'ixOnInactive' ],
  [ 'ixInactive', 'ixBorderInactive' ],
];

describe('themes', () => {
  it('should have sufficient distance between adjacent colors', () => {
    const violators: string[] = [];
    Object.entries(supportedThemes).forEach(([ name, theme ]) => {
      pairs.forEach(([ k1, k2 ]) => {
        expect(theme[k1]).toBeDefined();
        expect(theme[k2]).toBeDefined();
        expect(isColor(theme[k1])).toBe(true);
        expect(isColor(theme[k2])).toBe(true);
        const c1 = str2rgba(theme[k1] as string);
        const c2 = str2rgba(theme[k2] as string);
        if (maxColorDistance(c1, c2) < 30) {
          violators.push(`${name} ${k1} ${k2}`);
        }
      });
    });
    expect(violators).toEqual([]);
  });
});
