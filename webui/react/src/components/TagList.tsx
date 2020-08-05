import { PlusOutlined } from '@ant-design/icons';
import { Input, Tag, Tooltip } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import { alphanumericSorter } from 'utils/data';
import { toRem } from 'utils/dom';
import { toHtmlId, truncate } from 'utils/string';

import css from './TagList.module.scss';

const TAG_MAX_LENGTH = 20;
interface Props {
  className?: string;
  tags: string[];
  onChange?: (oldTag: string, newTag: string) => void;
  onCreate?: (tag: string) => void;
  onDelete?: (tag: string) => void;
}

const EditableTagList: React.FC<Props> = ({
  className, tags, onChange, onCreate, onDelete,
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
  const inputRef = useRef<Input>(null);
  const editInputRef = useRef<Input>(null);

  const handleClose = useCallback(removedTag => {
    if (onDelete) onDelete(removedTag);
  }, [ onDelete ]);

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
    if (onCreate && newTag && tags.indexOf(newTag) === -1) onCreate(newTag);
    setState(state => ({ ...state, inputValue: '', inputVisible: false }));
  }, [ onCreate, state, tags ]);

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
    if (onChange && oldTag && newTag && tags.indexOf(newTag) === -1) {
      onChange(oldTag, newTag);
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
  if (className) classes.push(className);

  return (
    <div className={classes.join(' ')} onClick={stopPropagation}>
      {tags
        .sort((a, b) => alphanumericSorter(a, b))
        .map((tag, index) => {
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
        <Tag className={css.tagPlus + ' tagPlus'} onClick={handleTagPlus}>
          <PlusOutlined /> New Tag
        </Tag>
      )}
    </div>
  );
};

export default EditableTagList;
