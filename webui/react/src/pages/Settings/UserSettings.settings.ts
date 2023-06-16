import { any } from 'io-ts';

import { SettingsConfig } from 'hooks/useSettings';

type Char =
  | 'A'
  | 'B'
  | 'C'
  | 'D'
  | 'E'
  | 'F'
  | 'G'
  | 'H'
  | 'I'
  | 'J'
  | 'K'
  | 'L'
  | 'M'
  | 'N'
  | 'O'
  | 'P'
  | 'Q'
  | 'R'
  | 'S'
  | 'T'
  | 'U'
  | 'V'
  | 'W'
  | 'X'
  | 'Y'
  | 'Z';

interface KeyboardShortcut {
  ctrl: boolean;
  meta: boolean;
  alt: boolean;
  shift: boolean;
  letter?: Char;
}

export interface Settings {
  navbarCollapsed: KeyboardShortcut;
}

const shortCutSettingsConfig: SettingsConfig<Settings> = {
  settings: {
    navbarCollapsed: {
      defaultValue: {
        alt: false,
        ctrl: false,
        letter: 'U',
        meta: true,
        shift: false,
      },
      skipUrlEncoding: true,
      storageKey: 'navbarCollapsed',
      type: any,
    },
  },
  storagePath: 'shortcuts',
};

export default shortCutSettingsConfig;

export const shortcutMet = (e: KeyboardEvent, shortcut: KeyboardShortcut): boolean =>
  e.ctrlKey === shortcut.ctrl &&
  e.metaKey === shortcut.meta &&
  e.altKey === shortcut.alt &&
  e.shiftKey === shortcut.shift &&
  e.code === `Key${shortcut.letter}`;

export const shortcutToString = (shortcut: KeyboardShortcut): string => {
  const os = window.navigator.userAgent;
  const s: string[] = [];
  shortcut.ctrl && s.push('Ctrl');
  shortcut.meta && s.push(os.includes('Mac') ? 'Cmd' : os.includes('Win') ? 'Win' : 'Super');
  shortcut.shift && s.push('Shift');
  shortcut.alt && s.push('Alt');
  shortcut.letter && s.push(shortcut.letter);
  return s.join(' + ');
};
