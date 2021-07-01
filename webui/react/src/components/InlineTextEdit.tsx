import React, { KeyboardEvent, useCallback, useRef, useState } from 'react';
import ContentEditable from 'react-contenteditable';

import Icon from 'components/Icon';
import { IndicatorUnpositioned } from 'components/Spinner';

import css from './InlineTextEdit.module.scss';

interface Props {
  setValue: (newValue: string) => void;
  value: string;
}

const InlineTextEdit: React.FC<Props> = ({ setValue, value }: Props) => {
  const [ isChanged, setIsChanged ] = useState(false);
  const [ isFocused, setIsFocused ] = useState(false);
  const [ isSaving, setIsSaving ] = useState(false);
  const innerValueRef = useRef<string>(value);
  const inputRef = React.useRef<HTMLElement>();

  const clear = useCallback(() => {
    innerValueRef.current = value;
    setIsChanged(false);
    setIsFocused(false);

    if (!inputRef.current) return;
    inputRef.current.blur();
  }, [ value ]);
  const focus = useCallback(() => {
    if (!inputRef.current) return;
    inputRef.current.focus();
  }, []);
  const save = useCallback(async () => {
    setIsSaving(true);
    setIsChanged(false);
    await setValue(innerValueRef.current);
    setIsSaving(false);
  }, [ setValue ]);

  const handleBlur = useCallback(() => {
    setIsFocused(false);
  }, []);
  const handleChange = useCallback(() => {
    if (!inputRef.current) return;
    innerValueRef.current = inputRef.current?.innerText;
    setIsChanged(innerValueRef.current !== value);
  }, [ value ]);
  const handleFocus = useCallback(() => {
    setIsFocused(true);
  }, []);
  const handleSetRef = useCallback((el: HTMLElement) => {
    inputRef.current = el;
  }, []);
  const handleKeyDown = useCallback((evt: KeyboardEvent<HTMLDivElement>) => {
    if ([ 'Esc', 'Escape' ].includes(evt.key)) {
      evt.preventDefault();
      clear();
    }
    if (evt.key === 'Enter') {
      evt.preventDefault();
      save();
    }
  }, [ clear, save ]);

  return (
    <>
      <ContentEditable
        autoCapitalize="false"
        autoCorrect="false"
        className={css.input}
        disabled={isSaving}
        html={innerValueRef.current}
        innerRef={handleSetRef}
        spellCheck="false"
        onBlur={handleBlur}
        onChange={handleChange}
        onFocus={handleFocus}
        onKeyDown={handleKeyDown}
      />
      {isSaving && (
        <IndicatorUnpositioned size="small" />
      )}
      {!isSaving && (isChanged || isFocused) && <>
        <span className={css.button} onClick={save}><Icon name="checkmark" size="small" /></span>
        <span className={css.button} onClick={clear}><Icon name="close" size="small" /></span>
      </>}
      {!isSaving && !isFocused && !isChanged && (
        <span className={css.button} onClick={focus}><Icon name="pencil" size="small" /></span>
      )}
    </>
  );
};

export default InlineTextEdit;
