import { CheckpointState, CommandState, ResourceState, RunState } from 'types';

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
}

export interface Theme {
  animationCurve: string;
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
    danger: {
      dark: string;
      light: string;
      normal: string;
    };
    monochrome: string[];
    overlay: string;
    states: Record<StateColors, string>;
  };
  focus: {
    shadow: string;
  };
  font: {
    family: string;
  };
  outline: string;
  shadow: string;
  sizes: {
    border: {
      radius: string;
      width: string;
    };
    font: {[size in ShirtSize]: string};
    /* eslint-disable @typescript-eslint/member-ordering */
    icon: {
      tiny: string;
      small: string;
      medium: string;
      large: string;
    };
    /* eslint-enable @typescript-eslint/member-ordering */
    layout: {[size in ShirtSize]: string};
    navigation: {
      maxWidth: string;
      minWidth: string;
      toolbarHeight: string;
      topbarHeight: string;
    };
  };
}

/*
 * When updating colors, update `variables.less` as well.
 * Currently two sources of truth due to Ant Design.
 */
export const lightTheme: Theme = {
  animationCurve: '0.2s cubic-bezier(0.785, 0.135, 0.15, 0.86)',
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
      inactive: '#666666',
      success: '#009900',
      suspended: '#cc9900',
    },
  },
  focus: { shadow: '0 0 0 0.2rem rgba(0, 155, 222, 0.25)' },
  font: { family: 'Objektiv Mk3' },
  outline: '0 0 0.4rem 0 #009bde',
  shadow: '0.2rem 0.2rem 0.4rem 0 rgba(0, 0, 0, 0.25)',
  sizes: {
    border: {
      radius: '0.4rem',
      width: '0.1rem',
    },
    /* eslint-disable sort-keys-fix/sort-keys-fix */
    font: {
      micro: '1.0rem',
      tiny: '1.1rem',
      small: '1.2rem',
      medium: '1.4rem',
      large: '1.6rem',
      big: '1.8rem',
      great: '2.0rem',
      huge: '2.2rem',
      enormous: '2.4rem',
      giant: '2.8rem',
      jumbo: '3.6rem',
      mega: '4rem',
    },
    icon: {
      tiny: '1.2rem',
      small: '1.6rem',
      medium: '2rem',
      large: '2.4rem',
    },
    layout: {
      micro: '0.2rem',
      tiny: '0.4rem',
      small: '0.6rem',
      medium: '0.8rem',
      large: '1.2rem',
      big: '1.6rem',
      great: '1.8rem',
      huge: '2rem',
      enormous: '2.4rem',
      giant: '3.2rem',
      jumbo: '3.6rem',
      mega: '4rem',
    },
    /* eslint-enable sort-keys-fix/sort-keys-fix */
    navigation: {
      maxWidth: '24rem',
      minWidth: '5.6rem',
      toolbarHeight: '5.6rem',
      topbarHeight: '5.6rem',
    },
  },
};

const stateColorMapping = {
  [RunState.Active]: 'active',
  [RunState.Canceled]: 'inactive',
  [RunState.Completed]: 'success',
  [RunState.Deleted]: 'failed',
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
};

export const getStateColorCssVar = (
  state: RunState | CommandState | ResourceState | CheckpointState | undefined,
): string => {
  const name = state ? stateColorMapping[state] : 'active';
  return `var(--theme-colors-states-${name})`;
};

export const getStateColor = (
  state: RunState | CommandState | ResourceState,
): string => {
  const name = state ? stateColorMapping[state] : 'active';
  return window.getComputedStyle(document.body).getPropertyValue(`--theme-colors-states-${name}`);
};

export enum ThemeId {
  Light = 'light',
  Dark = 'dark',
}

export const defaultThemeId: ThemeId = ThemeId.Light;

export default {
  [ThemeId.Dark]: lightTheme,
  [ThemeId.Light]: lightTheme,
};
