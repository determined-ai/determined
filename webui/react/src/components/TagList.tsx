import { PlusOutlined } from '@ant-design/icons';
import { Input, Tag, Tooltip } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import Link from 'components/Link';
import { alphaNumericSorter } from 'utils/sort';
import { toHtmlId, truncate } from 'utils/string';

import css from './TagList.module.scss';

interface Props {
  compact?: boolean;
  ghost?: boolean;
  onChange?: (tags: string[]) => void;
  tags: string[];
}

export const ARIA_LABEL_CONTAINER = 'new-tag-container';
export const ARIA_LABEL_TRIGGER = 'new-tag-trigger';
export const ARIA_LABEL_INPUT = 'new-tag-input';

const TAG_MAX_LENGTH = 10;
const COMPACT_MAX_THRESHOLD = 4;

const EditableTagList: React.FC<Props> = (
  { compact, ghost, tags, onChange }: Props,
) => {
  const initialState = {
    editInputIndex: -1,
    editInputValue: '',
    editOldInputValue: '',
    inputValue: '',
    inputVisible: false,
    inputWidth: 82,
  };
  const [ state, setState ] = useState(initialState);
  const [ showMore, setShowMore ] = useState(false);
  const inputRef = useRef<Input>(null);
  const editInputRef = useRef<Input>(null);

  const handleClose = useCallback(removedTag => {
    onChange?.(tags.filter(tag => tag !== removedTag));
  }, [ onChange, tags ]);

  const handleTagPlus = useCallback(() => {
    setState(state => ({ ...state, inputVisible: true }));
  }, []);

  useEffect(() => {
    if (state.inputVisible) inputRef.current?.focus();
  }, [ state.inputVisible ]);

  useEffect(() => {
    if (state.editInputIndex === -1) return;
    editInputRef.current?.focus();
    editInputRef.current?.select();
  }, [ state.editInputIndex ]);

  const stopPropagation = useCallback((e: React.MouseEvent) => e.stopPropagation(), []);

  const handleInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    e.persist();
    setState(state => ({ ...state, inputValue: e.target?.value }));
  }, []);

  const handleInputConfirm = useCallback(() => {
    const { inputValue } = state;
    const newTag = inputValue.trim();
    if (newTag && tags.indexOf(newTag) === -1) {
      onChange?.([ newTag, ...tags ]);
    }
    setState(state => ({ ...state, inputValue: '', inputVisible: false }));
  }, [ onChange, state, tags ]);

  const handleEditInputFocus = useCallback((e: React.FocusEvent<HTMLInputElement>) => {
    e.persist();
    setState(state => ({ ...state, editOldInputValue: e.target?.value }));
  }, []);

  const handleEditInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    e.persist();
    setState(state => ({ ...state, editInputValue: e.target?.value }));
  }, []);

  const handleEditInputConfirm = useCallback(() => {
    const { editInputValue, editOldInputValue } = state;
    const oldTag = editOldInputValue.trim();
    const newTag = editInputValue.trim();
    if (oldTag && newTag) {
      const updatedTags = tags.filter(tag => tag !== oldTag);
      if (updatedTags.indexOf(newTag) === -1) {
        updatedTags.push(newTag);
      }
      onChange?.(updatedTags);
    }
    setState(state => ({
      ...state,
      editInputIndex: -1,
      editInputValue: '',
      editOldInputValue: '',
    }));
  }, [ onChange, state, tags ]);

  const { editInputIndex, editInputValue, inputVisible, inputValue, inputWidth } = state;

  const classes = [ css.base ];
  if (ghost) classes.push(css.ghost);

  return (
    <div aria-label={ARIA_LABEL_CONTAINER} className={classes.join(' ')} onClick={stopPropagation}>
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
                key={tag}
                ref={editInputRef}
                size="small"
                style={{ width: inputWidth }}
                value={editInputValue}
                width={inputWidth}
                onBlur={handleEditInputConfirm}
                onChange={handleEditInputChange}
                onFocus={handleEditInputFocus}
                onPressEnter={handleEditInputConfirm} />
            );
          }

          const htmlId = toHtmlId(tag);
          const isLongTag = tag.length > TAG_MAX_LENGTH;

          const tagElement = (
            <Tag
              className={css.tagEdit}
              closable={true}
              id={htmlId}
              key={tag}
              onClose={() => handleClose(tag)}>
              <span
                onClick={e => {
                  e.preventDefault();
                  const element = document.getElementById(htmlId);
                  const rect = element?.getBoundingClientRect();
                  setState(state => ({
                    ...state,
                    editInputIndex: index,
                    editInputValue: tag,
                    inputWidth: rect?.width || state.inputWidth,
                  }));
                }}>
                {isLongTag ? truncate(tag, TAG_MAX_LENGTH) : tag}
              </span>
            </Tag>
          );
          return isLongTag ? <Tooltip key={tag} title={tag}>{tagElement}</Tooltip> : tagElement;
        })}
      {inputVisible ? (
        <Input
          aria-label={ARIA_LABEL_INPUT}
          className={css.tagInput}
          ref={inputRef}
          size="small"
          style={{ width: inputWidth }}
          type="text"
          value={inputValue}
          onBlur={handleInputConfirm}
          onChange={handleInputChange}
          onPressEnter={handleInputConfirm} />
      ) : (
        <Tag aria-label={ARIA_LABEL_TRIGGER} className={css.tagPlus} onClick={handleTagPlus}>
          <PlusOutlined /> New Tag
        </Tag>
      )}
    </div>
  );
};

export default EditableTagList;
