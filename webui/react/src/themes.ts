import { CommandState, RunState } from 'types';

export enum ShirtSize {
  micro = 'micro',
  tiny = 'tiny',
  small = 'small',
  medium = 'medium',
  big = 'big',
  large = 'large',
  jumbo = 'jumbo',
  huge = 'huge',
  giant = 'giant',
}

export interface Theme {
  colors: {
    core: {
      action: string;
      primary: string;
      secondary: string;
      tertiary: string;
    };
    monochrome: string[];
    states: {
      active: string;
      failed: string;
      inactive: string;
      success: string;
      suspended: string;
    };
  };
  font: {
    family: string;
  };
  shadow: string;
  sizes: {
    border: {
      radius: string;
      width: string;
    };
    font: {[size in ShirtSize]: string};
    icon: {
      tiny: string;
      small: string;
      medium: string;
      large: string;
    };
    layout: {[size in ShirtSize]: string};
    navbar: {
      height: string;
    };
    sidebar: {
      minWidth: string;
      maxWidth: string;
    };
  };
}

export const lightTheme: Theme = {
  colors: {
    core: {
      action: '#009bde',
      primary: '#f77b21',
      secondary: '#0d1e2b',
      tertiary: '#234b65',
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
      '#f2f2f2',  // 14 -  Platinum
      '#f7f7f7',  // 15 - Dusty
      '#fafafa',  // 16 - Fog
      '#ffffff',  // 17 - White
    ],
    states: {
      active: '#009bde',
      failed: '#cc0000',
      inactive: '#666666',
      success: '#009900',
      suspended: '#cc9900',
    },
  },
  font: {
    family: 'Objektiv Mk3',
  },
  shadow: '0.2rem 0.2rem 0.4rem 0 #cccccc',
  sizes: {
    border: {
      radius: '0.4rem',
      width: '0.1rem',
    },
    /* eslint-disable sort-keys */
    font: {
      micro: '1.0rem',
      tiny: '1.1rem',
      small: '1.2rem',
      medium: '1.4rem',
      big: '1.6rem',
      large: '1.8rem',
      jumbo: '2.0rem',
      huge: '2.4rem',
      giant: '3.6rem',
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
      big: '1.6rem',
      large: '2rem',
      jumbo: '2.4rem',
      huge: '3.2rem',
      giant: '3.6rem',
    },
    /* eslint-enable sort-keys */
    navbar: {
      height: '4.8rem',
    },
    sidebar: {
      maxWidth: '14.9rem',
      minWidth: '4.8rem',
    },
  },
};

export const getStateColor = (state: RunState | CommandState | undefined): string => {
  const stateColorMapping = {
    [RunState.Active]: 'var(--theme-colors-states-active)',
    [RunState.Canceled]: 'var(--theme-colors-states-inactive)',
    [RunState.Completed]: 'var(--theme-colors-states-success)',
    [RunState.Errored]: 'var(--theme-colors-states-failed)',
    [RunState.Paused]: 'var(--theme-colors-states-suspended)',
    [RunState.StoppingCanceled]: 'var(--theme-colors-states-inactive)',
    [RunState.StoppingCompleted]: 'var(--theme-colors-states-success)',
    [RunState.StoppingError]: 'var(--theme-colors-states-failed)',
    [CommandState.Pending]: 'var(--theme-colors-states-suspended)',
    [CommandState.Assigned]: 'var(--theme-colors-states-suspended)',
    [CommandState.Pulling]: 'var(--theme-colors-states-active)',
    [CommandState.Starting]: 'var(--theme-colors-states-active)',
    [CommandState.Running]: 'var(--theme-colors-states-active)',
    [CommandState.Terminating]: 'var(--theme-colors-states-inactive)',
    [CommandState.Terminated]: 'var(--theme-colors-states-inactive)',
  };

  return stateColorMapping[state || RunState.Active];
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
