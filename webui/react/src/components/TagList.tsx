import { PlusOutlined } from '@ant-design/icons';
import { Input, Tag, Tooltip } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import { truncate } from 'utils/string';

import css from './TagList.module.scss';

const TAG_MAX_LENGTH = 20;
interface Props {
  tags: string[];
  setTags: (tags: string[]) => void;
  className?: string;
}

const EditableTagList: React.FC<Props> = ({ tags, setTags, className }: Props) => {
  const initialState = {
    editInputIndex: -1,
    editInputValue: '',
    inputValue: '',
    inputVisible: false,
  };
  const [ state, setState ] = useState(initialState);
  const inputRef = useRef<Input>(null);
  const editInputRef = useRef<Input>(null);

  const handleClose = useCallback(removedTag => {
    const newTags = tags.filter(tag => tag !== removedTag);
    setTags(newTags);
  }, [ tags, setTags ]);

  const handleTagPlus = useCallback(() => {
    setState(state => ({ ...state, inputVisible: true }));
  }, []);

  useEffect(() => {
    if (state.inputVisible) inputRef.current?.focus();
  }, [ state.inputVisible ]);

  useEffect(() => {
    if (state.editInputIndex !== -1) editInputRef.current?.focus();
  }, [ state.editInputIndex ]);

  const stopPropagation = useCallback( (e: React.MouseEvent) => e.stopPropagation(), []);

  const handleInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    e.persist();
    setState(state => ({ ...state, inputValue: e.target?.value }));
  }, []);

  const handleInputConfirm = useCallback(() => {
    const { inputValue } = state;
    if (inputValue && tags.indexOf(inputValue) === -1) {
      const newTags = [ ...tags, inputValue ];
      setTags(newTags);
    }
    setState(state => ({ ...state, inputValue: '', inputVisible: false }));
  }, [ setTags, state, tags ]);

  const handleEditInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    e.persist();
    setState(state => ({ ...state, editInputValue: e.target?.value }));
  }, []);

  const handleEditInputConfirm = useCallback(() => {
    const { editInputIndex, editInputValue } = state;
    if (editInputValue && editInputIndex > -1) {
      const newTags = [ ...tags ];
      newTags[editInputIndex] = editInputValue;
      setTags(newTags);
    }
    setState(state => ({ ...state, editInputIndex: -1, editInputValue: '' }));
  }, [ setTags, state, tags ]);

  const { inputVisible, inputValue, editInputIndex, editInputValue } = state;

  const classes = [ css.base ];
  if (className) classes.push(className);

  return (
    <div className={classes.join(' ')} onClick={stopPropagation}>
      {tags.map((tag, index) => {
        if (editInputIndex === index) {
          return (
            <Input
              className={css.tagInput}
              key={tag}
              ref={editInputRef}
              size="small"
              value={editInputValue}
              onBlur={handleEditInputConfirm}
              onChange={handleEditInputChange}
              onPressEnter={handleEditInputConfirm}
            />
          );
        }

        const isLongTag = tag.length > TAG_MAX_LENGTH;

        const tagElem = (
          <Tag
            className={css.editTag}
            closable={true}
            key={tag}
            onClose={() => handleClose(tag)}
          >
            <span
              onDoubleClick={e => {
                e.preventDefault();
                setState(state => ({ ...state, editInputIndex: index, editInputValue: tag }));
              }}
            >
              {isLongTag ? truncate(tag, TAG_MAX_LENGTH) : tag}
            </span>
          </Tag>
        );
        return isLongTag ? (
          <Tooltip key={tag} title={tag}>
            {tagElem}
          </Tooltip>
        ) : (
          tagElem
        );
      })}
      {inputVisible && (
        <Input
          className={css.tagInput}
          ref={inputRef}
          size="small"
          type="text"
          value={inputValue}
          onBlur={handleInputConfirm}
          onChange={handleInputChange}
          onPressEnter={handleInputConfirm}
        />
      )}
      {!inputVisible && (
        <Tag className={css.tagPlus + ' tagPlus'} onClick={handleTagPlus}>
          <PlusOutlined /> New Tag
        </Tag>
      )}
    </div>
  );
};

export default EditableTagList;
