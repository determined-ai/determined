import React, {
  ChangeEvent,
  HTMLAttributes,
  KeyboardEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';

import Spinner from 'shared/components/Spinner/Spinner';

import css from './InlineEditor.module.scss';

interface Props extends HTMLAttributes<HTMLDivElement> {
  allowClear?: boolean;
  allowNewline?: boolean;
  disabled?: boolean;
  focusSignal?: number;
  maxLength?: number;
  onCancel?: () => void;
  onSave?: (newValue: string) => Promise<Error | void>;
  pattern?: RegExp;
  placeholder?: string;
  value: string;
}

const CODE_ENTER = 'Enter';
const CODE_ESCAPE = 'Escape';

const InlineEditor: React.FC<Props> = ({
  allowClear = true,
  allowNewline = false,
  disabled = false,
  pattern = new RegExp(''),
  maxLength,
  placeholder,
  value,
  onCancel,
  onSave,
  focusSignal,
  ...props
}: Props) => {
  const growWrapRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [currentValue, setCurrentValue] = useState<string>(value);
  const [isEditable, setIsEditable] = useState<boolean>(false);
  const [isSaving, setIsSaving] = useState<boolean>(false);
  const [isInvalidValue, setIsInvalidValue] = useState<boolean>(false);
  const classes: string = useMemo(() => {
    const classSet = new Set([css.base]);
    if (isEditable) classSet.add(css.editable);
    if (isSaving) classSet.add(css.loading);
    if (maxLength && currentValue && currentValue.length === maxLength) {
      classSet.add(css.maxLength);
    }
    if (disabled) classSet.add(css.disabled);

    return `${Array.from(classSet).join(' ')} ${isInvalidValue ? css.shakeAnimation : ''}`;
  }, [currentValue, disabled, isEditable, isInvalidValue, isSaving, maxLength]);

  useEffect(() => {
    if (focusSignal != null && !disabled) {
      setIsEditable(true);
      textareaRef.current?.focus();
    }
  }, [focusSignal, setIsEditable, disabled]);

  const updateEditorValue = useCallback(
    (value: string) => {
      let newValue = value;
      if (maxLength) newValue = newValue.slice(0, maxLength);
      if (textareaRef.current) textareaRef.current.value = newValue;
      if (growWrapRef.current) growWrapRef.current.dataset.value = newValue || placeholder;
      setCurrentValue(newValue);
    },
    [maxLength, placeholder],
  );

  const cancel = useCallback(() => {
    updateEditorValue(value);
    if (onCancel) onCancel();
  }, [onCancel, updateEditorValue, value]);

  const save = useCallback(
    async (newValue: string) => {
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
    },
    [onSave, pattern, updateEditorValue, value],
  );

  const handleWrapperClick = useCallback(() => {
    if (disabled) return;
    setIsEditable(true);
  }, [disabled]);

  /*
   * To trigger a save or cancel, we trigger the blur.
   * It is considered a save if the value has changed
   * and not empty, otherwise we assume it is a cancel.
   */
  const handleTextareaBlur = useCallback(() => {
    if (!textareaRef.current) return;

    const newValue = textareaRef.current.value.trim();
    (!!newValue || allowClear) && newValue !== value ? save(newValue) : cancel();

    // Reset `isEditable` to false if the blur was user triggered.
    setIsEditable(false);
  }, [allowClear, cancel, save, value]);

  const handleTextareaChange = useCallback(
    (e: ChangeEvent<HTMLTextAreaElement>) => {
      setIsInvalidValue(false);
      const textarea = e.target as HTMLTextAreaElement;
      let newValue = textarea.value;
      if (!allowNewline) newValue = newValue.replace(/(\r?\n|\r\n?)/g, '');
      updateEditorValue(newValue);
    },
    [allowNewline, updateEditorValue],
  );

  const handleTextareaKeyPress = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      if (!isEditable) {
        e.preventDefault();
        return;
      }
      if (e.which === 229) {
        // Ignore keydown event until IME is confirmed like Japanese and Chinese
        return;
      }
      if (e.code === CODE_ENTER && (!allowNewline || !e.shiftKey)) {
        e.preventDefault();
      }
      if (e.code === CODE_ESCAPE) {
        // Restore the original value upon escape key.
        updateEditorValue(value);
        setIsEditable(false);
      } else if (!e.shiftKey && e.code === CODE_ENTER) {
        setIsEditable(false);
      }
    },
    [allowNewline, isEditable, updateEditorValue, value],
  );

  useEffect(() => {
    updateEditorValue(value);
  }, [updateEditorValue, value]);

  useEffect(() => {
    if (!textareaRef.current || document.activeElement !== textareaRef.current) return;
    isEditable ? textareaRef.current.focus() : textareaRef.current.blur();
  }, [isEditable]);

  return (
    <div className={classes} {...props}>
      <div className={css.growWrap} ref={growWrapRef} onClick={handleWrapperClick}>
        <textarea
          cols={1}
          disabled={disabled}
          maxLength={maxLength}
          placeholder={placeholder}
          ref={textareaRef}
          rows={1}
          onBlur={handleTextareaBlur}
          onChange={handleTextareaChange}
          onKeyDown={handleTextareaKeyPress}
        />
        <div className={css.backdrop} />
        <div className={css.spinner}>
          <Spinner spinning={isSaving} />
        </div>
      </div>
    </div>
  );
};

export default InlineEditor;
