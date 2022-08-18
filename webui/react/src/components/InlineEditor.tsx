import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Spinner from 'shared/components/Spinner/Spinner';

import css from './InlineEditor.module.scss';

interface Props extends React.HTMLAttributes<HTMLDivElement> {
  allowClear?: boolean;
  allowNewline?: boolean;
  allowSpellCheck?: boolean;
  disabled?: boolean;
  focusSignal?: number;
  maxLength?: number;
  onCancel?: () => void;
  onSave?: (newValue: string) => Promise<Error|void>;
  pattern?: RegExp;
  placeholder?: string;
  resizable?: 'both' | 'horizontal' | 'vertical' | 'none',
  value: string;
}

const CODE_ENTER = 'Enter';
const CODE_ESCAPE = 'Escape';

const InlineEditor: React.FC<Props> = ({
  allowClear = true,
  allowNewline = false,
  allowSpellCheck = true,
  disabled = false,
  pattern = new RegExp(''),
  resizable = 'none',
  maxLength,
  placeholder,
  value,
  onCancel,
  onSave,
  focusSignal,
  ...props
}: Props) => {
  const editorRef = useRef<HTMLDivElement>(null);
  const [ currentValue, setCurrentValue ] = useState<string>(value);
  const [ isEditable, setIsEditable ] = useState<boolean>(false);
  const [ isSaving, setIsSaving ] = useState<boolean>(false);
  const [ isInvalidValue, setIsInvalidValue ] = useState<boolean>(false);
  const [ isMaxLength, setIsMaxLength ] = useState<boolean>(false);
  const classes: string = useMemo(() => {
    const classSet = new Set([ css.base ]);
    classSet.add(isEditable ? css.editable : css.base);
    classSet.add(isSaving ? css.loading : css.base);
    classSet.add(isMaxLength ? css.maxLength : css.base);
    classSet.add(disabled ? css.disabled : css.base);

    return `${Array.from(classSet).join(' ')} ${isInvalidValue ? css.shakeAnimation : ''}`;
  }, [ disabled, isEditable, isInvalidValue, isMaxLength, isSaving ]);

  useEffect(() => {
    if (focusSignal && !disabled) {
      setIsEditable(true);
      editorRef.current?.focus();
    }
  }, [ focusSignal, setIsEditable, disabled ]);

  const updateEditorValue = useCallback((value: string) => {
    let newValue = value;
    if (maxLength) { newValue = newValue.slice(0, maxLength); }
    if (editorRef.current) { editorRef.current.innerText = newValue; }
    // setCurrentValue(newValue);
  }, [ maxLength ]);

  const cancel = useCallback(() => {
    updateEditorValue(value);
    if (onCancel) { onCancel(); }
  }, [ onCancel, updateEditorValue, value ]);

  const save = useCallback(async (newValue: string) => {
    if (!pattern.test(newValue)) {
      setIsInvalidValue(true);
      updateEditorValue(value);
      return;
    }
    if (onSave) {
      setIsSaving(true);
      const err = await onSave(newValue);
      if (err != null) {
        updateEditorValue(value);
      }
      setIsSaving(false);
    }
  }, [ onSave, pattern, updateEditorValue, value ]);

  const handleWrapperClick = useCallback(() => {
    if (disabled) { return; }
    setIsEditable(true);
  }, [ disabled ]);

  /*
   * To trigger a save or cancel, we trigger the blur.
   * It is considered a save if the value has changed
   * and not empty, otherwise we assume it is a cancel.
   */
  const handleTextareaBlur = useCallback(() => {
    if (!editorRef.current) { return; }

    const newValue = allowNewline ? currentValue.trimRight() : currentValue.trim();

    if ((!!newValue || allowClear) && newValue !== value) {
      save(newValue);
    } else {
      cancel();
    }

    // Reset `isEditable` to false if the blur was user triggered.
    setIsEditable(false);
  }, [ allowClear, allowNewline, cancel, currentValue, save, value ]);

  const handleEditorInput = useCallback((e: React.ChangeEvent<HTMLDivElement>) => {
    setIsInvalidValue(false);
    setCurrentValue(e.target.innerText);
  }, []);

  const handleTextareaKeyDown = useCallback((e: React.KeyboardEvent<HTMLDivElement>) => {
    if (!isEditable || e.which === 229) {
      // e.which is to ignore keydown event until IME is confirmed like Japanese and Chinese
      e.preventDefault();
      return;
    }
    if (maxLength && editorRef.current && editorRef.current.innerText.length >= maxLength) {
      if (!e.key.includes('Arrow') && e.key !== 'Backspace' && !e.metaKey) e.preventDefault();
      setIsMaxLength(true);
      return;
    } else {
      setIsMaxLength(false);
    }
    if (e.code === CODE_ENTER && (!(allowNewline && e.shiftKey))) e.preventDefault();
    if (e.code === CODE_ESCAPE) {
      // Restore the original value upon escape key.
      updateEditorValue(value);
      setIsEditable(false);
    } else if (!e.shiftKey && e.code === CODE_ENTER) {
      setIsEditable(false);
    }
  }, [ allowNewline, isEditable, maxLength, updateEditorValue, value ]);

  useEffect(() => {
    updateEditorValue(value);
  }, [ updateEditorValue, value ]);

  useEffect(() => {
    if (!editorRef.current || document.activeElement !== editorRef.current) { return; }
    isEditable ? editorRef.current.focus() : editorRef.current.blur();
  }, [ isEditable ]);

  return (
    <div className={classes} onClick={handleWrapperClick} {...props}>
      <div
        className={css.textEditor}
        contentEditable={!disabled}
        dir="auto"
        placeholder={placeholder}
        ref={editorRef}
        role="textbox"
        spellCheck={allowSpellCheck}
        style={{ resize: resizable }}
        onBlur={handleTextareaBlur}
        onInput={handleEditorInput}
        onKeyDown={handleTextareaKeyDown}
      />
      <div className={css.spinner}>
        <Spinner className={css.spinner} spinning={isSaving} />
      </div>
    </div>
  );
};

export default InlineEditor;
