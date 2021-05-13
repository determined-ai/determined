import { PlusOutlined } from '@ant-design/icons';
import { Input, Tag, Tooltip } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import Link from 'components/Link';
import { toRem } from 'utils/dom';
import { alphanumericSorter } from 'utils/sort';
import { toHtmlId, truncate } from 'utils/string';

import css from './TagList.module.scss';

interface Props {
  compact?: boolean;
  onChange?: (tags: string[]) => void;
  tags: string[];
}

const TAG_MAX_LENGTH = 20;
const COMPACT_MAX_THRESHOLD = 3;

const EditableTagList: React.FC<Props> = ({
  compact,
  tags,
  onChange,
}: Props) => {
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
    if (onChange) {
      onChange(tags.filter(tag => tag !== removedTag));
    }
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

  const stopPropagation = useCallback( (e: React.MouseEvent) => e.stopPropagation(), []);

  const handleInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    e.persist();
    setState(state => ({ ...state, inputValue: e.target?.value }));
  }, []);

  const handleInputConfirm = useCallback(() => {
    const { inputValue } = state;
    const newTag = inputValue.trim();
    if (onChange && newTag && tags.indexOf(newTag) === -1) {
      onChange([ newTag, ...tags ]);
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
    if (onChange && oldTag && newTag) {
      const updatedTags = tags.filter(tag => tag !== oldTag);
      if (updatedTags.indexOf(newTag) === -1) {
        updatedTags.push(newTag);
      }
      onChange(updatedTags);
    }
    setState(state => ({
      ...state,
      editInputIndex: -1,
      editInputValue: '',
      editOldInputValue: '',
    }));
  }, [ onChange, state, tags ]);

  const { editInputIndex, editInputValue, inputVisible, inputValue, inputWidth } = state;

  return (
    <div className={css.base} onClick={stopPropagation}>
      {tags
        .sort((a, b) => alphanumericSorter(a, b))
        .map((tag, index) => {
          if (compact && !showMore && index >= COMPACT_MAX_THRESHOLD) {
            if (index > COMPACT_MAX_THRESHOLD) return null;
            return (
              <Link key="more" onClick={() => setShowMore(true)}>
                +{tags.length - COMPACT_MAX_THRESHOLD} more
              </Link>
            );
          }
          if (editInputIndex === index) {
            return (
              <Input
                className={css.tagInput}
                key={tag}
                ref={editInputRef}
                size="small"
                style={{ width: toRem(inputWidth) }}
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
          className={css.tagInput}
          ref={inputRef}
          size="small"
          style={{ width: toRem(inputWidth) }}
          type="text"
          value={inputValue}
          onBlur={handleInputConfirm}
          onChange={handleInputChange}
          onPressEnter={handleInputConfirm} />
      ) : (
        <Tag className={css.tagPlus} onClick={handleTagPlus}>
          <PlusOutlined /> New Tag
        </Tag>
      )}
    </div>
  );
};

export default EditableTagList;
