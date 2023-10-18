import { Tag } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import Icon from 'components/kit/Icon';
import Input, { InputRef } from 'components/kit/Input';
import { alphaNumericSorter, toHtmlId, truncate } from 'components/kit/internal/functions';
import { ValueOf } from 'components/kit/internal/types';
import Tooltip from 'components/kit/Tooltip';

import css from './Tags.module.scss';
export const TagAction = {
  Add: 'Add',
  Remove: 'Remove',
  Update: 'Update',
} as const;

export type TagAction = ValueOf<typeof TagAction>;

export interface Props {
  compact?: boolean;
  disabled?: boolean;
  ghost?: boolean;
  // UpdatedId refers to index now, should change this to tag ID in the future.
  onAction?: (action: TagAction, tag: string, updatedId?: number) => void;
  tags: string[];
}

export const ARIA_LABEL_CONTAINER = 'new-tag-container';
export const ARIA_LABEL_TRIGGER = 'new-tag-trigger';
export const ARIA_LABEL_INPUT = 'new-tag-input';

const TAG_MAX_LENGTH = 50;
const COMPACT_MAX_THRESHOLD = 6;

const Tags: React.FC<Props> = ({ compact, disabled = false, ghost, tags, onAction }: Props) => {
  const initialState = {
    editInputIndex: -1,
    inputVisible: false,
    inputWidth: 82,
  };
  const [state, setState] = useState(initialState);
  const [showMore, setShowMore] = useState(false);
  const inputRef = useRef<InputRef>(null);
  const editInputRef = useRef<InputRef>(null);

  const handleClose = useCallback(
    (removedTag: string) => {
      onAction?.(TagAction.Remove, removedTag);
    },
    [onAction],
  );

  const handleTagPlus = useCallback(() => {
    setState((state) => ({ ...state, inputVisible: true }));
  }, []);

  useEffect(() => {
    if (state.inputVisible) inputRef.current?.focus();
  }, [state.inputVisible]);

  useEffect(() => {
    if (state.editInputIndex === -1) return;
    editInputRef.current?.focus();
    editInputRef.current?.select();
  }, [state.editInputIndex]);

  const stopPropagation = useCallback((e: React.MouseEvent) => e.stopPropagation(), []);

  const handleInputConfirm = (
    e: React.FocusEvent | React.KeyboardEvent,
    previousValue?: string,
    tagID?: number,
  ) => {
    const newTag = (e.target as HTMLInputElement).value.trim();
    const oldTag = previousValue?.trim();
    if (newTag) {
      if (oldTag && newTag !== oldTag) {
        onAction?.(TagAction.Update, newTag, tagID);
      } else {
        onAction?.(TagAction.Add, newTag);
      }
    }
    setState((state) => ({ ...state, editInputIndex: -1, inputVisible: false }));
  };

  const { editInputIndex, inputVisible, inputWidth } = state;

  const classes = [css.base];
  if (ghost) classes.push(css.ghost);

  const addTagControls = inputVisible ? (
    <Input
      aria-label={ARIA_LABEL_INPUT}
      className={css.tagInput}
      defaultValue=""
      ref={inputRef}
      size="small"
      style={{ width: inputWidth }}
      type="text"
      onBlur={handleInputConfirm}
      onPressEnter={handleInputConfirm}
    />
  ) : (
    !disabled && (
      <Tag aria-label={ARIA_LABEL_TRIGGER} className={css.tagPlus} onClick={handleTagPlus}>
        <Icon decorative name="add" size="tiny" /> Add Tag
      </Tag>
    )
  );

  return (
    <div aria-label={ARIA_LABEL_CONTAINER} className={classes.join(' ')} onClick={stopPropagation}>
      {compact && addTagControls}
      {tags
        .sort((a, b) => alphaNumericSorter(a, b))
        .map((tag, index) => {
          if (compact && !showMore && index >= COMPACT_MAX_THRESHOLD) {
            if (index > COMPACT_MAX_THRESHOLD) return null;
            return (
              <a className={css.showMore} key="more" onClick={() => setShowMore(true)}>
                +{tags.length - COMPACT_MAX_THRESHOLD} more
              </a>
            );
          }
          if (editInputIndex === index) {
            return (
              <Input
                aria-label={ARIA_LABEL_INPUT}
                className={css.tagInput}
                defaultValue={tag}
                key={tag}
                ref={editInputRef}
                size="small"
                style={{ width: inputWidth }}
                onBlur={(e) => handleInputConfirm(e, tag, index)}
                onPressEnter={(e) => handleInputConfirm(e, tag, index)}
              />
            );
          }

          const htmlId = toHtmlId(tag);
          const isLongTag: boolean = tag.length > TAG_MAX_LENGTH;

          const tagElement = (
            <Tag closable={!disabled} id={htmlId} key={tag} onClose={() => handleClose(tag)}>
              <span
                onClick={(e) => {
                  e.preventDefault();
                  if (disabled) return;
                  const element = document.getElementById(htmlId);
                  const rect = element?.getBoundingClientRect();
                  setState((state) => ({
                    ...state,
                    editInputIndex: index,
                    inputWidth: rect?.width ?? state.inputWidth,
                  }));
                }}>
                {isLongTag && !disabled ? truncate(tag, TAG_MAX_LENGTH) : tag}
              </span>
            </Tag>
          );
          return isLongTag && !compact ? (
            <Tooltip content={tag} key={tag}>
              {tagElement}
            </Tooltip>
          ) : (
            tagElement
          );
        })}
      {!compact && addTagControls}
    </div>
  );
};

export default Tags;

// Eventually we will deprecate API calls that take the updated list of tags, and replace them with API calls that take only the updated tag. At that point, the tagsActionHelper could be removed or revised.
export const tagsActionHelper = (
  tags: string[],
  callbackFn: (tags: string[]) => void,
): ((action: TagAction, tag: string, updatedId?: number) => void) => {
  return (action: TagAction, tag: string, updatedId?: number) => {
    let newTags = [...tags];
    if (action === TagAction.Add) {
      newTags.push(tag);
    } else if (action === TagAction.Remove) {
      newTags = tags.filter((t) => t !== tag);
    } else if (action === TagAction.Update && updatedId !== undefined) {
      newTags[updatedId] = tag;
    }
    callbackFn(newTags);
  };
};
