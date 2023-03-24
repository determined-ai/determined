import { EditOutlined } from '@ant-design/icons';
import { Card, Space } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { unstable_useBlocker as useBlocker } from 'react-router-dom';

import Button from 'components/kit/Button';
import Input from 'components/kit/Input';
import Tooltip from 'components/kit/Tooltip';
import Spinner from 'shared/components/Spinner/Spinner';
import { ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

import Markdown from './Markdown';
import css from './NotesCard.module.scss';

interface Props {
  disabled?: boolean;
  extra?: React.ReactNode;
  noteChangeSignal?: number;
  notes: string;
  onChange?: (editedNotes: string) => void;
  onSave?: (editedNotes: string) => Promise<void>;
  onSaveTitle?: (editedTitle: string) => Promise<void>;
  style?: React.CSSProperties;
  title?: string;
}

const NotesCard: React.FC<Props> = ({
  disabled = false,
  notes,
  onSave,
  onSaveTitle,
  style,
  title = 'Notes',
  extra,
  onChange,
  noteChangeSignal,
}: Props) => {
  const [isEditing, setIsEditing] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [editedNotes, setEditedNotes] = useState(notes);

  const blocker = () => {
    if (isEditing && notes !== editedNotes) {
      const answer = window.confirm(
        'You have unsaved notes, are you sure you want to leave? Unsaved notes will be lost.',
      );
      return !answer;
    }
    return false;
  };
  useBlocker(() => blocker());

  const existingNotes = useRef(notes);

  useEffect(() => {
    existingNotes.current = notes;
  }, [notes]);

  useEffect(() => {
    setIsEditing(false);
    setIsLoading(false);
    setEditedNotes(existingNotes.current);
  }, [noteChangeSignal]);

  const editNotes = useCallback(() => {
    if (disabled) return;
    setIsEditing(true);
  }, [disabled]);

  const cancelEdit = useCallback(() => {
    setIsEditing(false);
    setEditedNotes(notes);
  }, [notes]);

  const saveNotes = useCallback(async () => {
    try {
      setIsLoading(true);
      await onSave?.(editedNotes.trim());
      setIsEditing(false);
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to update notes.',
        silent: true,
        type: ErrorType.Api,
      });
    }
    setIsLoading(false);
  }, [editedNotes, onSave]);

  const handleEditedNotes = useCallback(
    (newNotes: string) => {
      setEditedNotes(newNotes);
      onChange?.(newNotes);
    },
    [onChange],
  );

  const handleNotesClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.detail > 1 || notes === '') editNotes();
    },
    [editNotes, notes],
  );

  useEffect(() => {
    setEditedNotes(notes);
    setIsEditing(false);
  }, [notes]);

  return (
    <Card
      bodyStyle={{
        flexGrow: 1,
        flexShrink: 1,
        overflow: 'auto',
        padding: 0,
      }}
      className={css.base}
      extra={
        isEditing ? (
          <Space size="small">
            <Button size="small" onClick={cancelEdit}>
              Cancel
            </Button>
            <Button size="small" type="primary" onClick={saveNotes}>
              Save
            </Button>
          </Space>
        ) : (
          disabled || (
            <Space size="middle">
              <Tooltip title="Edit">
                <EditOutlined onClick={editNotes} />
              </Tooltip>
              {extra}
            </Space>
          )
        )
      }
      headStyle={{ marginTop: '16px', minHeight: 'fit-content', paddingInline: '16px' }}
      style={{ ...style }}
      title={
        <Input
          defaultValue={title}
          disabled={!onSaveTitle || disabled}
          onBlur={(e) => {
            const newValue = e.currentTarget.value;
            onSaveTitle?.(newValue);
          }}
          onPressEnter={(e) => {
            e.currentTarget.blur();
          }}
        />
      }>
      <Spinner spinning={isLoading}>
        <Markdown
          disabled={disabled}
          editing={isEditing}
          markdown={isEditing ? editedNotes : notes}
          onChange={handleEditedNotes}
          onClick={handleNotesClick}
        />
      </Spinner>
    </Card>
  );
};

export default NotesCard;
