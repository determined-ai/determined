import React, {
  ChangeEvent, HTMLAttributes, KeyboardEvent, useCallback, useEffect, useRef, useState,
} from 'react';

import css from './InlineEditor.module.scss';
import Spinner from './Spinner';

interface Props extends HTMLAttributes<HTMLDivElement> {
  allowClear?: boolean;
  allowNewline?: boolean;
  disabled?: boolean;
  isOnDark?: boolean;
  maxLength?: number;
  onCancel?: () => void;
  onSave?: (newValue: string) => Promise<void>;
  placeholder?: string;
  value: string;
}

const CODE_ENTER = 'Enter';
const CODE_ESCAPE = 'Escape';

const InlineEditor: React.FC<Props> = ({
  allowClear = true,
  allowNewline = false,
  disabled = false,
  isOnDark = false,
  maxLength,
  placeholder,
  value,
  onCancel,
  onSave,
  ...props
}: Props) => {
  const growWrapRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [ currentValue, setCurrentValue ] = useState(value);
  const [ isEditable, setIsEditable ] = useState(false);
  const [ isSaving, setIsSaving ] = useState(false);
  const classes = [ css.base ];

  if (isOnDark) classes.push(css.onDark);
  if (isEditable) classes.push(css.editable);
  if (isSaving) classes.push(css.loading);
  if (maxLength && currentValue && currentValue.length === maxLength) {
    classes.push(css.maxLength);
  }
  if (disabled) classes.push(css.disabled);

  const updateEditorValue = useCallback((value: string) => {
    let newValue = value;
    if (maxLength) newValue = newValue.slice(0, maxLength);
    if (textareaRef.current) textareaRef.current.value = newValue;
    if (growWrapRef.current) growWrapRef.current.dataset.value = newValue;
    setCurrentValue(newValue);
  }, [ maxLength ]);

  const cancel = useCallback(() => {
    updateEditorValue(value);
    if (onCancel) onCancel();
  }, [ onCancel, updateEditorValue, value ]);

  const save = useCallback(async (newValue: string) => {
    if (onSave) {
      try {
        setIsSaving(true);
        await onSave(newValue);
      } finally {
        setIsSaving(false);
      }
    }
  }, [ onSave ]);

  const handleWrapperClick = useCallback(() => {
    if (disabled) return;
    setIsEditable(true);
  }, [ disabled ]);

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
  }, [ allowClear, cancel, save, value ]);

  const handleTextareaChange = useCallback((e: ChangeEvent<HTMLTextAreaElement>) => {
    const textarea = e.target as HTMLTextAreaElement;
    let newValue = textarea.value;
    if (!allowNewline) newValue = newValue.replace(/(\r?\n|\r\n?)/g, '');
    updateEditorValue(newValue);
  }, [ allowNewline, updateEditorValue ]);

  const handleTextareaKeyPress = useCallback((e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (!isEditable) {
      e.preventDefault();
      return;
    }
    if (e.code === CODE_ENTER) {
      if (!allowNewline || !e.shiftKey) e.preventDefault();
    }
  }, [ allowNewline, isEditable ]);

  const handleTextareaKeyUp = useCallback((e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.code === CODE_ESCAPE) {
      // Restore the original value upon escape key.
      updateEditorValue(value);
      setIsEditable(false);
    } else if (!e.shiftKey && e.code === CODE_ENTER) {
      setIsEditable(false);
    }
  }, [ updateEditorValue, value ]);

  useEffect(() => {
    updateEditorValue(value);
  }, [ updateEditorValue, value ]);

  useEffect(() => {
    if (!textareaRef.current || document.activeElement !== textareaRef.current) return;
    isEditable ? textareaRef.current.focus() : textareaRef.current.blur();
  }, [ isEditable ]);

  return (
    <div className={classes.join(' ')} {...props}>
      <div className={css.growWrap} ref={growWrapRef} onClick={handleWrapperClick}>
        <textarea
          maxLength={maxLength}
          placeholder={placeholder}
          readOnly={!isEditable}
          ref={textareaRef}
          rows={1}
          onBlur={handleTextareaBlur}
          onChange={handleTextareaChange}
          onKeyPress={handleTextareaKeyPress}
          onKeyUp={handleTextareaKeyUp}
        />
        <div className={css.spinner}>
          <Spinner spinning={isSaving} />
        </div>
      </div>
    </div>
  );
};

export default InlineEditor;
