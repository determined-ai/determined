/* eslint-disable @typescript-eslint/member-ordering */
/* eslint-disable sort-keys-fix/sort-keys-fix */
import {
  BrandingType, CheckpointState, CommandState, JobState, ResourceState, RunState, SlotState,
} from 'types';
import { isColor, rgba2str, rgbaFromGradient, str2rgba } from 'utils/color';

/*
 * Where did we get our sizes from?
 * https://www.quora.com/What-is-the-difference-among-big-large-huge-enormous-and-giant
 */
export enum ShirtSize {
  micro = 'micro',
  tiny = 'tiny',
  small = 'small',
  medium = 'medium',
  large = 'large',
  big = 'big',
  great = 'great',
  huge = 'huge',
  enormous = 'enormous',
  giant = 'giant',
  jumbo = 'jumbo',
  mega = 'mega',
}

enum StateColors {
  active = 'active',
  failed = 'failed',
  inactive = 'inactive',
  success = 'success',
  suspended = 'suspended',
  free = 'free',
  pending = 'pending',
  potential = 'potential'
}

export interface ThemeX {
  colors: {
    action: {
      dark: string;
      light: string;
      normal: string;
    };
    base: string[];
    brand: {
      dark: string;
      light: string;
      normal: string;
    };
    core: {
      background: string;
      secondary: string;
    },
    danger: {
      dark: string;
      light: string;
      normal: string;
    };
    monochrome: string[];
    overlay: string;
    states: Record<StateColors, string>;
    underlay: string;
  };
  focus: {
    shadow: string;
  };
  outline: string;
  shadow: string;
}

/*
 * When updating colors, update `variables.less` as well.
 * Currently two sources of truth due to Ant Design.
 */
const lightDeterminedTheme: ThemeX = {
  colors: {
    action: {
      dark: '#0088cc',
      light: '#00b2ff',
      normal: '#009bde',
    },
    base: [
      '#0d1e2b',  // 0 - A
      '#132231',  // 1 - B
      '#1f2d3c',  // 2 - C
      '#273847',  // 3 - D
      '#334251',  // 4 - E
      '#3d4c5a',  // 5 - F
      '#4A5764',  // 6 - G
      '#55626e',  // 7 - H
      '#616c78',  // 8 - I
      '#6B7681',  // 9 - J
      '#77818b',  // 10 - K
      '#828c95',  // 11 - L
      '#8d959e',  // 12 - M
    ],
    brand: {
      dark: '#ee6600',
      light: '#ff9933',
      normal: '#f77b21',
    },
    core: {
      background: '#f7f7f7',
      secondary: '#234B65',
    },
    danger: {
      dark: '#aa0000',
      light: '#ee0000',
      normal: '#cc0000',
    },
    monochrome: [
      '#000000',  // 0 - Black
      '#080808',  // 1 - Space
      '#141414',  // 2 - Jet
      '#1d1d1d',  // 3 - Vanta
      '#2b2b2b',  // 4 - Midnight
      '#333333',  // 5 - Onyx
      '#444444',  // 6 - Charcoal
      '#666666',  // 7 - Lead
      '#888888',  // 8 - Anchor
      '#aaaaaa',  // 9 - Grey
      '#bbbbbb',  // 10 - Rhino
      '#cccccc',  // 11 - Stainless Steel
      '#dddddd',  // 12 - Ash
      '#ececec',  // 13 - Silver
      '#f2f2f2',  // 14 - Platinum
      '#f7f7f7',  // 15 - Dusty
      '#fafafa',  // 16 - Fog
      '#ffffff',  // 17 - White
    ],
    overlay: 'rgba(255, 255, 255, 0.75)',
    states: {
      active: '#009bde',
      failed: '#cc0000',
      free: '#eee',
      inactive: '#666666',
      pending: '#6666cc',
      potential: '#ffffff',
      success: '#009900',
      suspended: '#cc9900',
    },
    underlay: 'rgba(0, 0, 0, 0.65)',
  },
  focus: { shadow: '0 0 0 2px rgba(0, 155, 222, 0.25)' },
  outline: '0 0 4px 0 #009bde',
  shadow: '2px 2px 4px 0 rgba(0, 0, 0, 0.25)',
};

const darkDeterminedTheme: ThemeX = {
  colors: {
    action: {
      dark: '#0088cc',
      light: '#00b2ff',
      normal: '#009bde',
    },
    base: [
      '#0d1e2b',  // 0 - A
      '#132231',  // 1 - B
      '#1f2d3c',  // 2 - C
      '#273847',  // 3 - D
      '#334251',  // 4 - E
      '#3d4c5a',  // 5 - F
      '#4A5764',  // 6 - G
      '#55626e',  // 7 - H
      '#616c78',  // 8 - I
      '#6B7681',  // 9 - J
      '#77818b',  // 10 - K
      '#828c95',  // 11 - L
      '#8d959e',  // 12 - M
    ],
    brand: {
      dark: '#ee6600',
      light: '#ff9933',
      normal: '#f77b21',
    },
    core: {
      background: '#141414',
      secondary: '#234B65',
    },
    danger: {
      dark: '#aa0000',
      light: '#ee0000',
      normal: '#cc0000',
    },
    monochrome: [
      '#ffffff',  // 17 - White
      '#fafafa',  // 16 - Fog
      '#f7f7f7',  // 15 - Dusty
      '#f2f2f2',  // 14 - Platinum
      '#ececec',  // 13 - Silver
      '#dddddd',  // 12 - Ash
      '#cccccc',  // 11 - Stainless Steel
      '#bbbbbb',  // 10 - Rhino
      '#aaaaaa',  // 9 - Grey
      '#888888',  // 8 - Anchor
      '#666666',  // 7 - Lead
      '#444444',  // 6 - Charcoal
      '#333333',  // 5 - Onyx
      '#2b2b2b',  // 4 - Midnight
      '#1d1d1d',  // 3 - Vanta
      '#141414',  // 2 - Jet
      '#080808',  // 1 - Space
      '#000000',  // 0 - Black
    ],
    overlay: 'rgba(255, 255, 255, 0.75)',
    states: {
      active: '#009bde',
      failed: '#cc0000',
      free: '#eee',
      inactive: '#666666',
      pending: '#6666cc',
      potential: '#ffffff',
      success: '#009900',
      suspended: '#cc9900',
    },
    underlay: 'rgba(0, 0, 0, 0.65)',
  },
  focus: { shadow: '0 0 0 2px rgba(0, 155, 222, 0.25)' },
  outline: '0 0 4px 0 #009bde',
  shadow: '2px 2px 4px 0 rgba(0, 0, 0, 0.25)',
};

const stateColorMapping = {
  [RunState.Active]: 'active',
  [RunState.Canceled]: 'inactive',
  [RunState.Completed]: 'success',
  [RunState.Deleted]: 'failed',
  [RunState.Deleting]: 'failed',
  [RunState.DeleteFailed]: 'failed',
  [RunState.Errored]: 'failed',
  [RunState.Paused]: 'suspended',
  [RunState.StoppingCanceled]: 'inactive',
  [RunState.StoppingCompleted]: 'success',
  [RunState.StoppingError]: 'failed',
  [RunState.Unspecified]: 'inactive',
  [CommandState.Pending]: 'suspended',
  [CommandState.Assigned]: 'suspended',
  [CommandState.Pulling]: 'active',
  [CommandState.Starting]: 'active',
  [CommandState.Running]: 'active',
  [CommandState.Terminating]: 'inactive',
  [CommandState.Terminated]: 'inactive',
  [ResourceState.Unspecified]: 'inactive',
  [SlotState.Free]: 'free',
  [SlotState.Pending]: 'pending',
  [SlotState.Running]: 'active',
  [SlotState.Potential]: 'potential',
  [JobState.SCHEDULED]: 'active',
  [JobState.SCHEDULEDBACKFILLED]: 'active',
  [JobState.QUEUED]: 'suspended',
};

export type StateOfUnion = RunState | CommandState | ResourceState | CheckpointState |
SlotState | JobState

export const getStateColorCssVar = (state: StateOfUnion | undefined): string => {
  const name = state ? stateColorMapping[state] : 'active';
  return `var(--theme-colors-states-${name})`;
};

export const getStateColor = (state: StateOfUnion | undefined): string => {
  const cssVar = getStateColorCssVar(state);
  return window.getComputedStyle(document.body).getPropertyValue(cssVar);
};

const generateStrongWeak = (theme: Theme): Theme => {
  const rgbaStrong = str2rgba(theme.strong);
  const rgbaWeak = str2rgba(theme.weak);

  for (const [ key, value ] of Object.entries(theme)) {
    const matches = key.match(/^(.+)(Strong|Weak)$/);
    if (matches?.length === 3 && value === undefined) {
      const isStrong = matches[2] === 'Strong';
      const baseKey = matches[1] as keyof Theme;
      const baseValue = theme[baseKey];
      if (baseValue && isColor(baseValue)) {
        const rgba = str2rgba(baseValue);
        const mixer = isStrong ? rgbaStrong : rgbaWeak;
        theme[key as keyof Theme] = rgba2str(rgbaFromGradient(rgba, mixer, 0.1));
      }
    }
  }
  return theme as Theme;
};

const themeBase = {
  // Font styles.
  fontFamily: '"Objektiv Mk3", Arial, Helvetica, sans-serif',
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

  // Specialized and unique styles.
  overlay: undefined,
  overlayStrong: undefined,
  overlayWeak: undefined,
  borderRadius: '4px',
  borderRadiusStrong: '8px',
  borderRadiusWeak: '2px',
  strokeWidth: '1px',
  strokeWidthStrong: '3px',
  strokeWidthWeak: '0.5px',
  elevation: undefined,
  elevationStrong: undefined,
  elevationWeak: undefined,

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
  statusWarning: 'rgba(204, 153, 0, 1.0)',
  statusWarningStrong: undefined,
  statusWarningWeak: undefined,
  statusWarningOn: 'rgba(255, 255, 255, 1.0)',
  statusWarningOnStrong: undefined,
  statusWarningOnWeak: undefined,
};

const themeLight = {
  // Palette colors for strong/weak calculations.
  strong: 'rgba(0, 0, 0, 1.0)',
  weak: 'rgba(255, 255, 255, 1.0)',

  // Area and surface styles.
  background: 'rgba(204, 204, 202, 1.0)',
  backgroundOn: 'rgba(18, 18, 18, 1.0)',
  backgroundBorder: undefined,
  stage: 'rgba(220, 219, 217, 1.0)',
  stageOn: 'rgba(69, 69, 69, 1.0)',
  stageBorder: 'rgba(194, 194, 194, 1.0)',
  surface: 'rgba(200, 199, 197, 1.0)',
  surfaceOn: 'rgba(0, 8, 16, 1.0)',
  surfaceBorder: 'rgba(190, 190, 190, 1.000)',
  float: 'rgba(195, 195, 195, 1.0)',
  floatOn: 'rgba(49, 49, 49, 1.0)',
  floatBorder: undefined,

  // Specialized and unique styles.
  overlay: 'rgba(255, 255, 255, 0.75)',
  overlayStrong: 'rgba(255, 255, 255, 1.0)',
  overlayWeak: 'rgba(255, 255, 255, 0.5)',
  elevation: '0px 6px 12px rgba(0, 0, 0, 0.12)',
  elevationStrong: '0px 12px 24px rgba(0, 0, 0, 0.12)',
  elevationWeak: '0px 2px 4px rgba(0, 0, 0, 0.24)',
};

const themeDark = {
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
  surface: 'rgba(55, 56, 57, 1.0)',
  surfaceOn: 'rgba(255, 247, 239, 1.0)',
  surfaceBorder: 'rgba(65, 65, 65, 1.000)',
  float: 'rgba(60, 60, 60, 1.0)',
  floatOn: 'rgba(206, 206, 206, 1.0)',
  floatBorder: undefined,

  // Specialized and unique styles.
  overlay: 'rgba(0, 0, 0, 0.75)',
  overlayStrong: 'rgba(0, 0, 0, 1.0)',
  overlayWeak: 'rgba(0, 0, 0, 0.5)',
  elevation: '0px 6px 12px rgba(255, 255, 255, 0.12)',
  elevationStrong: '0px 12px 24px rgba(255, 255, 255, 0.12)',
  elevationWeak: '0px 2px 4px rgba(255, 255, 255, 0.24)',
};

const themeLightDetermined: Theme = generateStrongWeak(Object.assign(themeBase, themeLight));
const themeDarkDetermined: Theme = generateStrongWeak(Object.assign(themeBase, themeDark));

const themeHpe = { brand: 'rgba(1, 169, 130, 1.0)' };
const themeLightHpe: Theme = generateStrongWeak(Object.assign(themeBase, themeLight, themeHpe));
const themeDarkHpe: Theme = generateStrongWeak(Object.assign(themeBase, themeDark, themeHpe));

export type Theme = Record<keyof typeof themeBase, string>;

export const globalCssVars = {
  animationCurve: '0.2s cubic-bezier(0.785, 0.135, 0.15, 0.86)',

  fontFamily: '"Objektiv Mk3", Arial, Helvetica, sans-serif',
  fontFamilyCode: '"Source Code Pro", monospace',

  iconTiny: '12px',
  iconSmall: '16px',
  iconMedium: '20px',
  iconLarge: '24px',
  iconBig: '28px',
  iconGreat: '32px',
  iconHuge: '36px',
  iconEnormous: '40px',
  iconGiant: '44px',
  iconJumbo: '48px',
  iconMega: '52px',

  navBottomBarHeight: '56px',
  navSideBarWidthMax: '240px',
  navSideBarWidthMin: '56px',
};

export enum DarkLight {
  Dark = 'dark',
  Light = 'light',
}

export default {
  [BrandingType.Determined]: {
    [DarkLight.Dark]: themeDarkDetermined,
    [DarkLight.Light]: themeLightDetermined,
  },
  [BrandingType.HPE]: {
    [DarkLight.Dark]: themeDarkHpe,
    [DarkLight.Light]: themeLightHpe,
  },
};
