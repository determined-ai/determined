import * as t from 'io-ts';

export const KeyboardShortcut = t.type({
  alt: t.boolean,
  ctrl: t.boolean,
  key: t.string,
  meta: t.boolean,
  shift: t.boolean,
});

export type KeyboardShortcut = t.TypeOf<typeof KeyboardShortcut>;

export const EmptyKeyboardShortcut = {
  alt: false,
  ctrl: false,
  key: '',
  meta: false,
  shift: false,
};

export const matchesShortcut = (e: KeyboardEvent, shortcut: KeyboardShortcut): boolean =>
  e.ctrlKey === shortcut.ctrl &&
  e.metaKey === shortcut.meta &&
  e.altKey === shortcut.alt &&
  e.shiftKey === shortcut.shift &&
  e.key.toUpperCase() === shortcut.key.toUpperCase();

export const shortcutToString = (shortcut: KeyboardShortcut): string => {
  const os = window.navigator.userAgent;
  const s: string[] = [];
  shortcut.ctrl && s.push('Ctrl');
  shortcut.meta && s.push(os.includes('Mac') ? 'Cmd' : os.includes('Win') ? 'Win' : 'Super');
  shortcut.shift && s.push('Shift');
  shortcut.alt && s.push('Alt');
  shortcut.key &&
    !['Control', 'Meta', 'Alt', 'Shift'].includes(shortcut.key) &&
    s.push(shortcut.key.toUpperCase());
  return s.join(' + ');
};
