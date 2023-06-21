export interface KeyboardShortcut {
  ctrl: boolean;
  meta: boolean;
  alt: boolean;
  shift: boolean;
  key?: string;
}

export const matchesShortcut = (e: KeyboardEvent, shortcut: KeyboardShortcut): boolean =>
  e.ctrlKey === shortcut.ctrl &&
  e.metaKey === shortcut.meta &&
  e.altKey === shortcut.alt &&
  e.shiftKey === shortcut.shift &&
  e.key === shortcut.key;

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
