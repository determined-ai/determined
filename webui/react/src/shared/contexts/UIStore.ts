import { DarkLight, Mode, Theme } from '../themes';

export interface StateUI {
    chromeCollapsed: boolean;
    darkLight: DarkLight;
    isPageHidden: boolean;
    mode: Mode;
    showChrome: boolean;
    showSpinner: boolean;
    theme: Theme;
}

export const initUI: StateUI = {
    chromeCollapsed: false,
    darkLight: DarkLight.Light,
    isPageHidden: false,
    mode: Mode.System,
    showChrome: true,
    showSpinner: false,
    theme: {} as Theme,
};

export enum StoreActionUI {
    HideUIChrome = 'HideUIChrome',
    HideUISpinner = 'HideUISpinner',
    SetMode = 'SetMode',
    SetPageVisibility = 'SetPageVisibility',
    SetTheme = 'SetTheme',
    ShowUIChrome = 'ShowUIChrome',
    ShowUISpinner = 'ShowUISpinner',
}

export type ActionUI =
    | { type: StoreActionUI.HideUIChrome }
    | { type: StoreActionUI.HideUISpinner }
    | { type: StoreActionUI.SetMode; value: Mode }
    | { type: StoreActionUI.SetPageVisibility; value: boolean }
    | { type: StoreActionUI.SetTheme; value: { darkLight: DarkLight, theme: Theme } }
    | { type: StoreActionUI.ShowUIChrome }
    | { type: StoreActionUI.ShowUISpinner }
/**
 * return a part of the input state that should be updated.
 * @param state ui state
 * @param action
 * @returns
 */
export const reducerUI = (state: StateUI, action: ActionUI): Partial<StateUI> | void => {
    switch (action.type) {
        case StoreActionUI.HideUIChrome:
            if (!state.showChrome) return;
            return { showChrome: false };
        case StoreActionUI.HideUISpinner:
            if (!state.showSpinner) return;
            return { showSpinner: false };
        case StoreActionUI.SetMode:
            return { mode: action.value };
        case StoreActionUI.SetPageVisibility:
            return { isPageHidden: action.value };
        case StoreActionUI.SetTheme:
            return {
                darkLight: action.value.darkLight,
                theme: action.value.theme,
            };
        case StoreActionUI.ShowUIChrome:
            if (state.showChrome) return;
            return { showChrome: true };
        case StoreActionUI.ShowUISpinner:
            if (state.showSpinner) return;
            return { showSpinner: true };
        default:
            return;
    }
};
