import { CheckpointState, CommandState, ResourceState, RunState, SlotState } from 'types';

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
      big: string;
      great: string;
      huge: string;
      enormous: string;
      giant: string;
      jumbo: string;
      mega: string;
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
const lightDeterminedTheme: Theme = {
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
      success: '#009900',
      suspended: '#cc9900',
    },
    underlay: 'rgba(0, 0, 0, 0.65)',
  },
  focus: { shadow: '0 0 0 2px rgba(0, 155, 222, 0.25)' },
  font: { family: 'Objektiv Mk3' },
  outline: '0 0 4px 0 #009bde',
  shadow: '2px 2px 4px 0 rgba(0, 0, 0, 0.25)',
  sizes: {
    border: {
      radius: '4px',
      width: '1px',
    },
    /* eslint-disable sort-keys-fix/sort-keys-fix */
    font: {
      micro: '10px',
      tiny: '11px',
      small: '12px',
      medium: '14px',
      large: '16px',
      big: '18px',
      great: '20px',
      huge: '22px',
      enormous: '24px',
      giant: '28px',
      jumbo: '36px',
      mega: '40px',
    },
    icon: {
      tiny: '12px',
      small: '16px',
      medium: '20px',
      large: '24px',
      big: '28px',
      great: '32px',
      huge: '36px',
      enormous: '40px',
      giant: '44px',
      jumbo: '48px',
      mega: '52px',
    },
    layout: {
      micro: '2px',
      tiny: '4px',
      small: '6px',
      medium: '8px',
      large: '12px',
      big: '16px',
      great: '18px',
      huge: '20px',
      enormous: '24px',
      giant: '32px',
      jumbo: '36px',
      mega: '40px',
    },
    /* eslint-enable sort-keys-fix/sort-keys-fix */
    navigation: {
      maxWidth: '240px',
      minWidth: '56px',
      toolbarHeight: '56px',
      topbarHeight: '56px',
    },
  },
};

const lightHpeTheme: Theme = {
  ...lightDeterminedTheme,
  colors: {
    ...lightDeterminedTheme.colors,
    brand: {
      dark: '#009069',
      light: '#1bc39c',
      normal: '#01a982',
    },
  },
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
};

type States = RunState | CommandState | ResourceState | CheckpointState | SlotState

export const getStateColorCssVar = (state: States | undefined): string => {
  const name = state ? stateColorMapping[state] : 'active';
  return `var(--theme-colors-states-${name})`;
};

export const getStateColor = (state: States | undefined): string => {
  const cssVar = getStateColorCssVar(state);
  return window.getComputedStyle(document.body).getPropertyValue(cssVar);
};

export enum ThemeId {
  DarkDetermined = 'dark-determined',
  DarkHPE = 'dark-hpe',
  LightDetermined = 'light-determined',
  LightHPE = 'light-hpe',
}

export const defaultThemeId: ThemeId = ThemeId.LightDetermined;

export default {
  [ThemeId.DarkDetermined]: lightDeterminedTheme,
  [ThemeId.LightDetermined]: lightDeterminedTheme,
  [ThemeId.DarkHPE]: lightHpeTheme,
  [ThemeId.LightHPE]: lightHpeTheme,
};
