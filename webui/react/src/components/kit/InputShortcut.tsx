import { Input, InputRef } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import { formatKey, KeyboardShortcut, shortcutToString } from 'utils/shortcut';

import css from './InputShortcut.module.scss';

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
    <div className={css.shortcut_input_conatiner}>
      <Input
        placeholder={placeholder}
        ref={inputRef}
        value={inputValue}
        onKeyDown={onKeyDown}
        {...props}
      />
    </div>
  );
};

export default InputShortcut;
