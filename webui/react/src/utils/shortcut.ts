import * as t from 'io-ts';

export const KeyboardShortcut = t.type({
  alt: t.boolean,
  ctrl: t.boolean,
  key: t.string,
  meta: t.boolean,
  shift: t.boolean,
});

export type KeyboardShortcut = t.TypeOf<typeof KeyboardShortcut>;

export const matchesShortcut = (e: KeyboardEvent, shortcut: KeyboardShortcut): boolean =>
  e.ctrlKey === shortcut.ctrl &&
  e.metaKey === shortcut.meta &&
  e.altKey === shortcut.alt &&
  e.shiftKey === shortcut.shift &&
  formatKey(e.code, e.key) === shortcut.key;

export const shortcutToString = (shortcut: KeyboardShortcut): string => {
  const os = window.navigator.userAgent;
  const s: string[] = [];
  shortcut.ctrl && s.push('Ctrl');
  shortcut.meta && s.push(os.includes('Mac') ? 'Cmd' : os.includes('Win') ? 'Win' : 'Super');
  shortcut.shift && s.push('Shift');
  shortcut.alt && s.push('Alt');
  shortcut.key && s.push(shortcut.key);
  return s.join(' + ');
};

export const formatKey = (code: string, key: string): string => {
  if (code.startsWith('Digit')) return code.replace('Digit', '');
  switch (code) {
    case 'BracketLeft':
      return '[';
    case 'BracketRight':
      return ']';
    case 'Backslash':
      return '\\';
    case 'Semicolon':
      return ';';
    case 'Quote':
      return "'";
    case 'Comma':
      return ',';
    case 'Period':
      return '.';
    case 'Slash':
      return '/';
    case 'Space':
      return 'Space';
    default:
      if (['Control', 'Meta', 'Alt', 'Shift'].includes(key)) return '';
      return key.toUpperCase();
  }
};
