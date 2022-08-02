import * as themes from './themes';
import { isColor, labDistance, rgb2lab, str2rgba } from './utils/color';

const supportedThemes = {
  darkDet: themes.themeDarkDetermined,
  darkHpe: themes.themeDarkHpe,
  lightDet: themes.themeLightDetermined,
  lightHpe: themes.themeLightHpe,
};

type ThemeVarPair = (keyof themes.Theme)[];
const genPairs = (name: string): ThemeVarPair[] => {
  const base = [
    [ `${name}`, `${name}On` ],
    [ `${name}`, `${name}Border` ],
  ] as ThemeVarPair[];

  // same intentsity pairs.
  const strong = [ ...base.map(([ a, b ]) => [ `${a}Strong`, `${b}Strong` ]) ] as ThemeVarPair[];
  const weak = [ ...base.map(([ a, b ]) => [ `${a}Weak`, `${b}Weak` ]) ] as ThemeVarPair[];

  // stronger color on top of weaker color.
  const strongOnDefault = [ ...base.map(([ a, b ]) => [ `${a}`, `${b}Strong` ]) ] as ThemeVarPair[];
  const defaultOnWeak = [ ...base.map(([ a, b ]) => [ `${a}Weak`, `${b}` ]) ] as ThemeVarPair[];

  // weak colors on top of strong colors.
  // const weakOnDefault = [ ...base.map(([ a, b ]) => [ `${a}`, `${b}Weak` ]) ] as ThemeVarPair[];
  // const defaultOnStrong = [ ...base.map(([ a, b ]) =>
  //   [ `${a}Strong`, `${b}` ]) ] as ThemeVarPair[];

  // CHECK: which of these pairs make sens to test?
  return [ ...base,
    ...strong,
    ...weak,
    ...strongOnDefault,
    ...defaultOnWeak,
    // ...weakOnDefault,
    // ...defaultOnStrong,
  ];
};

/** pars of theme colors used next to eachother */
const pairs: ThemeVarPair[] = [
  [ 'ixActive', 'ixOnActive' ],
  [ 'ixActive', 'ixBorderActive' ],
  [ 'ixInactive', 'ixOnInactive' ],
  [ 'ixInactive', 'ixBorderInactive' ],
  [ 'background', 'backgroundOn' ],
];

describe('themes', () => {
  beforeAll(() => {
    const newPairs = [ 'surface', 'stage', 'float' ]
      .map(genPairs)
      .reduce((acc, cur) => acc.concat(cur), []);
    pairs.push(...newPairs);

  });

  it('should have sufficient distance between adjacent colors', () => {
    /** defines the required minimum distance between at least one of the rgba values. */
    const TOLERANCE = 4.1; // TODO: raise the bar.
    const violators: (string|number)[][] = [];
    Object.entries(supportedThemes).forEach(([ name, theme ]) => {
      pairs.forEach(([ k1, k2 ]) => {
        expect(theme[k1]).toBeDefined();
        expect(theme[k2]).toBeDefined();
        expect(isColor(theme[k1])).toBe(true);
        expect(isColor(theme[k2])).toBe(true);
        const c1 = str2rgba(theme[k1] as string);
        const c2 = str2rgba(theme[k2] as string);
        const c1CL = rgb2lab(c1);
        const c2CL = rgb2lab(c2);
        const distance = labDistance(c1CL, c2CL);
        if (distance < TOLERANCE) {
          violators.push([ `Theme: ${name} - ${k1} ${k2}`, distance ]);
        }
      });
    });
    // sort violators by distance.
    const reports = violators
      .sort((a, b) => (a[1] as number) - (b[1] as number))
      .map(([ name, distance ]) => `${name} - ${distance}`);
    expect(reports).toEqual([]);
  });
});
