/* eslint-disable sort-keys-fix/sort-keys-fix */
import { isColor, rgba2str, rgbaMix, str2rgba } from 'components/kit/internal/color';
import { ValueOf } from 'components/kit/internal/types';

const STRONG_WEAK_DELTA = 45;

const generateStrongWeak = (theme: Theme): Theme => {
  const rgbaStrong = str2rgba(theme.strong);
  const rgbaWeak = str2rgba(theme.weak);

  for (const [key, value] of Object.entries(theme)) {
    const matches = key.match(/^(.+)(Strong|Weak)$/);
    if (matches?.length === 3 && value === undefined) {
      const isStrong = matches[2] === 'Strong';
      const baseKey = matches[1] as keyof Theme;
      const baseValue = theme[baseKey];
      if (baseValue && isColor(baseValue)) {
        const rgba = str2rgba(baseValue);
        const mixer = isStrong ? rgbaStrong : rgbaWeak;
        theme[key as keyof Theme] = rgba2str(rgbaMix(rgba, mixer, STRONG_WEAK_DELTA));
      }
    }
  }
  return theme as Theme;
};

export interface ThemProps {
  colorScheme: React.CSSProperties['colorScheme'];
  fontFamily: React.CSSProperties['fontFamily'];
  fontFamilyVar: React.CSSProperties['fontFamily'];
  fontFamilyCode: string;
  strong: React.CSSProperties['color'];
  weak: React.CSSProperties['color'];
  brand: React.CSSProperties['color'];
  brandStrong: React.CSSProperties['color'];
  brandWeak: React.CSSProperties['color'];
  background: React.CSSProperties['color'];
  backgroundStrong: React.CSSProperties['color'];
  backgroundWeak: React.CSSProperties['color'];
  backgroundOn: React.CSSProperties['color'];
  backgroundOnStrong: React.CSSProperties['color'];
  backgroundOnWeak: React.CSSProperties['color'];
  backgroundBorder: React.CSSProperties['color'];
  backgroundBorderStrong: React.CSSProperties['color'];
  backgroundBorderWeak: React.CSSProperties['color'];
  stage: React.CSSProperties['color'];
  stageStrong: React.CSSProperties['color'];
  stageWeak: React.CSSProperties['color'];
  stageOn: React.CSSProperties['color'];
  stageOnStrong: React.CSSProperties['color'];
  stageOnWeak: React.CSSProperties['color'];
  stageBorder: React.CSSProperties['color'];
  stageBorderStrong: React.CSSProperties['color'];
  stageBorderWeak: React.CSSProperties['color'];
  surface: React.CSSProperties['color'];
  surfaceStrong: React.CSSProperties['color'];
  surfaceWeak: React.CSSProperties['color'];
  surfaceOn: React.CSSProperties['color'];
  surfaceOnStrong: React.CSSProperties['color'];
  surfaceOnWeak: React.CSSProperties['color'];
  surfaceBorder: React.CSSProperties['color'];
  surfaceBorderStrong: React.CSSProperties['color'];
  surfaceBorderWeak: React.CSSProperties['color'];
  float: React.CSSProperties['color'];
  floatStrong: React.CSSProperties['color'];
  floatWeak: React.CSSProperties['color'];
  floatOn: React.CSSProperties['color'];
  floatOnStrong: React.CSSProperties['color'];
  floatOnWeak: React.CSSProperties['color'];
  floatBorder: React.CSSProperties['color'];
  floatBorderStrong: React.CSSProperties['color'];
  floatBorderWeak: React.CSSProperties['color'];
  ix: React.CSSProperties['color'];
  ixStrong: React.CSSProperties['color'];
  ixWeak: React.CSSProperties['color'];
  ixActive: React.CSSProperties['color'];
  ixInactive: React.CSSProperties['color'];
  ixOn: React.CSSProperties['color'];
  ixOnStrong: React.CSSProperties['color'];
  ixOnWeak: React.CSSProperties['color'];
  ixOnActive: React.CSSProperties['color'];
  ixOnInactive: React.CSSProperties['color'];
  ixBorder: React.CSSProperties['color'];
  ixBorderStrong: React.CSSProperties['color'];
  ixBorderWeak: React.CSSProperties['color'];
  ixBorderActive: React.CSSProperties['color'];
  ixBorderInactive: React.CSSProperties['color'];
  density: number;
  targetFocus: React.CSSProperties['boxShadow'];
  borderRadius: React.CSSProperties['borderRadius'];
  borderRadiusStrong: React.CSSProperties['borderRadius'];
  borderRadiusWeak: React.CSSProperties['borderRadius'];
  strokeWidth: React.CSSProperties['strokeWidth'];
  strokeWidthStrong: React.CSSProperties['strokeWidth'];
  strokeWidthWeak: React.CSSProperties['strokeWidth'];
  elevation: React.CSSProperties['boxShadow'];
  elevationStrong: React.CSSProperties['boxShadow'];
  elevationWeak: React.CSSProperties['boxShadow'];
  overlay: React.CSSProperties['color'];
  overlayStrong: React.CSSProperties['color'];
  overlayWeak: React.CSSProperties['color'];
  statusActive: React.CSSProperties['color'];
  statusActiveStrong: React.CSSProperties['color'];
  statusActiveWeak: React.CSSProperties['color'];
  statusActiveOn: React.CSSProperties['color'];
  statusActiveOnStrong: React.CSSProperties['color'];
  statusActiveOnWeak: React.CSSProperties['color'];
  statusCritical: React.CSSProperties['color'];
  statusCriticalStrong: React.CSSProperties['color'];
  statusCriticalWeak: React.CSSProperties['color'];
  statusCriticalOn: React.CSSProperties['color'];
  statusCriticalOnStrong: React.CSSProperties['color'];
  statusCriticalOnWeak: React.CSSProperties['color'];
  statusError: React.CSSProperties['color'];
  statusInactive: React.CSSProperties['color'];
  statusInactiveStrong: React.CSSProperties['color'];
  statusInactiveWeak: React.CSSProperties['color'];
  statusInactiveOn: React.CSSProperties['color'];
  statusInactiveOnStrong: React.CSSProperties['color'];
  statusInactiveOnWeak: React.CSSProperties['color'];
  statusPending: React.CSSProperties['color'];
  statusPendingStrong: React.CSSProperties['color'];
  statusPendingWeak: React.CSSProperties['color'];
  statusPendingOn: React.CSSProperties['color'];
  statusPendingOnStrong: React.CSSProperties['color'];
  statusPendingOnWeak: React.CSSProperties['color'];
  statusPotential: React.CSSProperties['color'];
  statusSuccess: React.CSSProperties['color'];
  statusSuccessStrong: React.CSSProperties['color'];
  statusSuccessWeak: React.CSSProperties['color'];
  statusSuccessOn: React.CSSProperties['color'];
  statusSuccessOnStrong: React.CSSProperties['color'];
  statusSuccessOnWeak: React.CSSProperties['color'];
  statusWarning: React.CSSProperties['color'];
  statusWarningStrong: React.CSSProperties['color'];
  statusWarningWeak: React.CSSProperties['color'];
  statusWarningOn: React.CSSProperties['color'];
  statusWarningOnStrong: React.CSSProperties['color'];
  statusWarningOnWeak: React.CSSProperties['color'];
}

const themeBase = {
  // Color schemes
  colorScheme: 'normal',

  // Font styles.
  fontFamily: 'Inter, Arial, Helvetica, sans-serif',
  fontFamilyVar: '"Inter var", Arial, Helvetica, sans-serif',
  fontFamilyCode: '"Source Code Pro", monospace',

  // Palette colors for strong/weak calculations.
  strong: undefined,
  weak: undefined,

  // Brand colors.
  brand: 'rgba(247, 123, 33, 1.0)',
  brandStrong: undefined,
  brandWeak: undefined,

  // Area and surface styles.
  background: undefined,
  backgroundStrong: undefined,
  backgroundWeak: undefined,
  backgroundOn: undefined,
  backgroundOnStrong: undefined,
  backgroundOnWeak: undefined,
  backgroundBorder: undefined,
  backgroundBorderStrong: undefined,
  backgroundBorderWeak: undefined,
  stage: undefined,
  stageStrong: undefined,
  stageWeak: undefined,
  stageOn: undefined,
  stageOnStrong: undefined,
  stageOnWeak: undefined,
  stageBorder: undefined,
  stageBorderStrong: undefined,
  stageBorderWeak: undefined,
  surface: undefined,
  surfaceStrong: undefined,
  surfaceWeak: undefined,
  surfaceOn: undefined,
  surfaceOnStrong: undefined,
  surfaceOnWeak: undefined,
  surfaceBorder: undefined,
  surfaceBorderStrong: undefined,
  surfaceBorderWeak: undefined,
  float: undefined,
  floatStrong: undefined,
  floatWeak: undefined,
  floatOn: undefined,
  floatOnStrong: undefined,
  floatOnWeak: undefined,
  floatBorder: undefined,
  floatBorderStrong: undefined,
  floatBorderWeak: undefined,

  // Interactive styles.
  ix: undefined,
  ixStrong: undefined,
  ixWeak: undefined,
  ixActive: undefined,
  ixInactive: undefined,
  ixOn: undefined,
  ixOnStrong: undefined,
  ixOnWeak: undefined,
  ixOnActive: undefined,
  ixOnInactive: undefined,
  ixBorder: undefined,
  ixBorderStrong: undefined,
  ixBorderWeak: undefined,
  ixBorderActive: undefined,
  ixBorderInactive: undefined,

  // Specialized and unique styles.
  density: '2',
  targetFocus: '0px 0px 4px rgba(0, 155, 222, 0.12)',
  borderRadius: '4px',
  borderRadiusStrong: '8px',
  borderRadiusWeak: '2px',
  strokeWidth: '1px',
  strokeWidthStrong: '3px',
  strokeWidthWeak: '0.5px',
  elevation: undefined,
  elevationStrong: undefined,
  elevationWeak: undefined,
  overlay: undefined,
  overlayStrong: undefined,
  overlayWeak: undefined,

  // Status styles.
  statusActive: 'rgba(0, 155, 222, 1.0)',
  statusActiveStrong: undefined,
  statusActiveWeak: undefined,
  statusActiveOn: 'rgba(255, 255, 255, 1.0)',
  statusActiveOnStrong: undefined,
  statusActiveOnWeak: undefined,
  statusCritical: 'rgba(204, 0, 0, 1.0)',
  statusCriticalStrong: undefined,
  statusCriticalWeak: undefined,
  statusCriticalOn: 'rgba(255, 255, 255, 1.0)',
  statusCriticalOnStrong: undefined,
  statusCriticalOnWeak: undefined,
  statusError: 'rgb(247, 140, 140)',
  statusInactive: 'rgba(102, 102, 102, 1.0)',
  statusInactiveStrong: undefined,
  statusInactiveWeak: undefined,
  statusInactiveOn: 'rgba(255, 255, 255, 1.0)',
  statusInactiveOnStrong: undefined,
  statusInactiveOnWeak: undefined,
  statusPending: 'rgba(102, 102, 204, 1.0)',
  statusPendingStrong: undefined,
  statusPendingWeak: undefined,
  statusPendingOn: 'rgba(255, 255, 255, 1.0)',
  statusPendingOnStrong: undefined,
  statusPendingOnWeak: undefined,
  statusPotential: 'rgba(255, 255, 255, 0)',
  statusSuccess: 'rgba(0, 153, 0, 1.0)',
  statusSuccessStrong: undefined,
  statusSuccessWeak: undefined,
  statusSuccessOn: 'rgba(255, 255, 255, 1.0)',
  statusSuccessOnStrong: undefined,
  statusSuccessOnWeak: undefined,
  statusWarning: 'rgba(204, 153, 0, 1.0)',
  statusWarningStrong: undefined,
  statusWarningWeak: undefined,
  statusWarningOn: 'rgba(255, 255, 255, 1.0)',
  statusWarningOnStrong: undefined,
  statusWarningOnWeak: undefined,
};

const themeLight = {
  // Color schemes
  colorScheme: 'light',

  // Palette colors for strong/weak calculations.
  strong: 'rgba(0, 0, 0, 1.0)',
  weak: 'rgba(255, 255, 255, 1.0)',

  // Area and surface styles.
  background: 'rgba(240, 240, 240, 1.0)',
  backgroundOn: 'rgba(18, 18, 18, 1.0)',
  backgroundBorder: undefined,
  stage: 'rgba(246, 246, 246, 1.0)',
  stageOn: 'rgba(69, 69, 69, 1.0)',
  stageBorder: 'rgba(194, 194, 194, 1.0)',
  surface: 'rgba(250, 250, 250, 1.0)',
  surfaceOn: 'rgba(0, 8, 16, 1.0)',
  surfaceBorder: 'rgba(212, 212, 212, 1.0)',
  float: 'rgba(255, 255, 255, 1.0)',
  floatOn: 'rgba(49, 49, 49, 1.0)',
  floatBorder: 'rgba(225, 225, 225, 1.0)',

  // Interactive styles.
  ix: 'rgba(255, 255, 255, 1.0)',
  ixActive: 'rgba(231, 247, 255, 1.0)',
  ixInactive: 'rgba(245, 245, 245, 1.0)',
  ixOn: 'rgba(38, 38, 38, 1.0)',
  ixOnActive: 'rgba(0, 155, 222, 1.0)',
  ixOnInactive: 'rgba(217, 217, 217, 1.0)',
  ixBorder: 'rgba(217, 217, 217, 1.0)',
  ixBorderActive: 'rgba(0, 155, 222, 1.0)',
  ixBorderInactive: 'rgba(217, 217, 217, 1.0)',
  ixCancel: 'rgba(89,89,89,1)',

  // Specialized and unique styles.
  overlay: 'rgba(255, 255, 255, 0.75)',
  overlayStrong: 'rgba(255, 255, 255, 1.0)',
  overlayWeak: 'rgba(255, 255, 255, 0.5)',
  elevation: '0px 6px 12px rgba(0, 0, 0, 0.12)',
  elevationStrong: '0px 12px 24px rgba(0, 0, 0, 0.12)',
  elevationWeak: '0px 2px 4px rgba(0, 0, 0, 0.24)',
};

const themeDark = {
  // Color schemes
  colorScheme: 'dark',

  // Palette colors for strong/weak calculations.
  strong: 'rgba(255, 255, 255, 1.0)',
  weak: 'rgba(0, 0, 0, 1.0)',

  // Area and surface styles.
  background: 'rgba(21, 21, 23, 1.0)',
  backgroundOn: 'rgba(237, 237, 237, 1.0)',
  backgroundBorder: undefined,
  stage: 'rgba(35, 36, 38, 1.0)',
  stageOn: 'rgba(186, 186, 186, 1.0)',
  stageBorder: 'rgba(61, 61, 61, 1.0)',
  surface: 'rgba(48, 49, 50, 1.0)',
  surfaceOn: 'rgba(255, 247, 239, 1.0)',
  surfaceBorder: 'rgba(85, 85, 85, 1.0)',
  float: 'rgba(60, 61, 62, 1.0)',
  floatOn: 'rgba(206, 206, 206, 1.0)',
  floatBorder: 'rgba(90, 91, 92, 1.0)',

  // Interactive styles.
  ix: 'rgba(21, 21, 23, 1.0)',
  ixActive: 'rgba(17, 27, 38, 1.0)',
  ixInactive: 'rgba(49, 49, 49, 1.0)',
  ixOn: 'rgba(209, 209, 209, 1.0)',
  ixOnActive: 'rgba(23, 125, 220, 1.0)',
  ixOnInactive: 'rgba(80, 80, 80, 1.0)',
  ixBorder: 'rgba(67, 67, 67, 1.0)',
  ixBorderActive: 'rgba(23, 125, 220, 1.0)',
  ixBorderInactive: 'rgba(80, 80, 80, 1.0)',
  ixCancel: 'rgba(115,115,115,1)',

  // Specialized and unique styles.
  overlay: 'rgba(0, 0, 0, 0.75)',
  overlayStrong: 'rgba(0, 0, 0, 1.0)',
  overlayWeak: 'rgba(0, 0, 0, 0.5)',
  elevation: '0px 6px 12px rgba(255, 255, 255, 0.06)',
  elevationStrong: '0px 12px 24px rgba(255, 255, 255, 0.06)',
  elevationWeak: '0px 2px 4px rgba(255, 255, 255, 0.12)',
};

export const themeLightDetermined: Theme = generateStrongWeak(
  Object.assign({}, themeBase, themeLight),
);
export const themeDarkDetermined: Theme = generateStrongWeak(
  Object.assign({}, themeBase, themeDark),
);
const themeHpe = { brand: 'rgba(1, 169, 130, 1.0)' };

export const themeLightHpe: Theme = generateStrongWeak(
  Object.assign({}, themeBase, themeLight, themeHpe),
);
export const themeDarkHpe: Theme = generateStrongWeak(
  Object.assign({}, themeBase, themeDark, themeHpe),
);

export type Theme = Record<keyof typeof themeBase, string>;

export const globalCssVars = {
  animationCurve: '0.2s cubic-bezier(0.785, 0.135, 0.15, 0.86)',

  iconBig: '28px',
  iconEnormous: '40px',
  iconGiant: '44px',
  iconGreat: '32px',
  iconHuge: '36px',
  iconJumbo: '48px',
  iconLarge: '24px',
  iconMedium: '20px',
  iconMega: '52px',
  iconSmall: '16px',
  iconTiny: '12px',

  navBottomBarHeight: '56px',
  navSideBarWidthMax: '240px',
  navSideBarWidthMin: '56px',
};

export const Mode = {
  System: 'system',
  Light: 'light',
  Dark: 'dark',
} as const;

export type Mode = ValueOf<typeof Mode>;

/**
 * DarkLight is a resolved form of `Mode` where we figure out
 * what `Mode.System` should ultimate resolve to (`Dark` vs `Light).
 */
export const DarkLight = {
  Dark: 'dark',
  Light: 'light',
} as const;

export type DarkLight = ValueOf<typeof DarkLight>;

export const getCssVar = (name: string): string => {
  const varName = name.replace(/^(var\()?(.*?)\)?$/i, '$2');
  return window.getComputedStyle(document.body)?.getPropertyValue(varName);
};
