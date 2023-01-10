import { PlusOutlined } from '@ant-design/icons';
import { Input, Tag } from 'antd';
import type { InputRef } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import Tooltip from 'components/kit/Tooltip';
import Link from 'components/Link';
import { alphaNumericSorter } from 'shared/utils/sort';
import { toHtmlId, truncate } from 'shared/utils/string';

import css from './TagList.module.scss';

interface Props {
  compact?: boolean;
  disabled?: boolean;
  ghost?: boolean;
  onChange?: (tags: string[]) => void;
  tags: string[];
}

export const ARIA_LABEL_CONTAINER = 'new-tag-container';
export const ARIA_LABEL_TRIGGER = 'new-tag-trigger';
export const ARIA_LABEL_INPUT = 'new-tag-input';

const TAG_MAX_LENGTH = 50;
const COMPACT_MAX_THRESHOLD = 6;

const EditableTagList: React.FC<Props> = ({
  compact,
  disabled = false,
  ghost,
  tags,
  onChange,
}: Props) => {
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
      onChange?.(tags.filter((tag) => tag !== removedTag));
    },
    [onChange, tags],
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

  const handleInputConfirm = useCallback(
    (
      e: React.FocusEvent<HTMLInputElement> | React.KeyboardEvent<HTMLInputElement>,
      previousValue?: string,
    ) => {
      const newTag = (e.target as HTMLInputElement).value.trim();
      const oldTag = previousValue?.trim();
      const updatedTags = tags.filter((tag) => tag !== oldTag);
      if (newTag) {
        if (!updatedTags.includes(newTag)) {
          updatedTags.push(newTag);
        }
        onChange?.(updatedTags);
      }
      setState((state) => ({ ...state, editInputIndex: -1, inputVisible: false }));
    },
    [onChange, tags],
  );

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
        <PlusOutlined /> Add Tag
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
              <Link className={css.showMore} key="more" onClick={() => setShowMore(true)}>
                +{tags.length - COMPACT_MAX_THRESHOLD} more
              </Link>
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
                width={inputWidth}
                onBlur={(e) => handleInputConfirm(e, tag)}
                onPressEnter={(e) => handleInputConfirm(e, tag)}
              />
            );
          }

          const htmlId = toHtmlId(tag);
          const isLongTag = tag.length > TAG_MAX_LENGTH;

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
            <Tooltip key={tag} title={tag}>
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

export default EditableTagList;
