import { Input, InputRef } from 'antd';
import * as t from 'io-ts';
import React, { useCallback, useEffect, useRef, useState } from 'react';

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

interface InputShortcutProps {
  disabled?: boolean;
  onChange?: (k: KeyboardShortcut | undefined) => void;
  placeholder?: string;
  value?: KeyboardShortcut;
}

const InputShortcut: React.FC<InputShortcutProps> = ({
  placeholder = 'Press any key',
  value,
  onChange,
  ...props
}: InputShortcutProps) => {
  const inputRef = useRef<InputRef>(null);
  const [inputValue, setInputValue] = useState<string>();

  useEffect(() => {
    value && setInputValue(shortcutToString(value));
  }, [value]);

  const onKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      e.preventDefault();
      const keys: KeyboardShortcut = {
        alt: e.altKey,
        ctrl: e.ctrlKey,
        key: formatKey(e.code, e.key),
        meta: e.metaKey,
        shift: e.shiftKey,
      };
      value ? onChange?.(keys) : setInputValue(shortcutToString(keys));
    },
    [onChange, value],
  );

  return (
    <Input
      placeholder={placeholder}
      ref={inputRef}
      value={inputValue}
      onKeyDown={onKeyDown}
      {...props}
    />
  );
};

export default InputShortcut;
