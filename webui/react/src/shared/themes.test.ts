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
    /** defines the required minimum distance between at least one of the rgba values. */
    const TOLERANCE = 25;
    const violators: string[] = [];
    Object.entries(supportedThemes).forEach(([ name, theme ]) => {
      pairs.forEach(([ k1, k2 ]) => {
        expect(theme[k1]).toBeDefined();
        expect(theme[k2]).toBeDefined();
        expect(isColor(theme[k1])).toBe(true);
        expect(isColor(theme[k2])).toBe(true);
        const c1 = str2rgba(theme[k1] as string);
        const c2 = str2rgba(theme[k2] as string);
        const distance = maxColorDistance(c1, c2);
        if (distance < TOLERANCE) {
          violators.push(`Theme: ${name} - ${k1} ${k2}. Distance: ${distance}`);
        }
      });
    });
    expect(violators).toEqual([]);
  });
});
