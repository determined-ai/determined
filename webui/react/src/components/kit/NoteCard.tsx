import { EditOutlined } from '@ant-design/icons';
import { Card, Space } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { unstable_useBlocker as useBlocker } from 'react-router-dom';

import Button from 'components/kit/Button';
import Input from 'components/kit/Input';
import Markdown from 'components/kit/internal/Markdown';
import Spinner from 'components/kit/internal/Spinner/Spinner';
import { ErrorHandler, ErrorType, Note } from 'components/kit/internal/types';
import Tooltip from 'components/kit/Tooltip';

import css from './NoteCard.module.scss';

interface Props {
  disabled?: boolean;
  disableTitle?: boolean;
  extra?: React.ReactNode;
  noteChangeSignal?: number;
  note: Note;
  onChange?: (editedNotes: string) => void;
  onError: ErrorHandler;
  onSaveNote: (notes: Note) => Promise<void>;
}

const NoteCard: React.FC<Props> = ({
  disabled = false,
  disableTitle = false,
  note,
  extra,
  onChange,
  onError,
  onSaveNote,
  noteChangeSignal,
}: Props) => {
  const [isEditing, setIsEditing] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [editedNotes, setEditedNotes] = useState(note?.contents || '');
  const [editedTitle, setEditedTitle] = useState(note?.name || '');
  const [notes, title] = useMemo(() => [note?.contents || '', note?.name || ''], [note]);

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
  const existingTitle = useRef(title);

  useEffect(() => {
    existingNotes.current = notes;
  }, [notes]);
  useEffect(() => {
    existingTitle.current = title;
  }, [title]);

  useEffect(() => {
    setIsEditing(false);
    setIsLoading(false);
    setEditedNotes(existingNotes.current);
    setEditedTitle(existingTitle.current);
  }, [noteChangeSignal]);

  const editNotes = useCallback(() => {
    if (disabled) return;
    setIsEditing(true);
  }, [disabled]);

  const cancelEdit = useCallback(() => {
    setIsEditing(false);
    setEditedNotes(notes);
    onChange?.(notes);
    setEditedTitle(title);
  }, [notes, title, onChange]);

  const onSave = useCallback(
    async (editNotes: string) => {
      await onSaveNote({ contents: editNotes, name: editedTitle });
    },
    [onSaveNote, editedTitle],
  );

  const onSaveTitle = useCallback(
    async (editTitle: string) => {
      await onSaveNote({ contents: editedNotes, name: editTitle });
    },
    [onSaveNote, editedNotes],
  );

  const saveNotes = useCallback(async () => {
    try {
      setIsLoading(true);
      await onSave?.(editedNotes.trim());
      setIsEditing(false);
    } catch (e) {
      onError(e, {
        publicSubject: 'Unable to update notes.',
        silent: true,
        type: ErrorType.Api,
      });
    }
    setIsLoading(false);
  }, [editedNotes, onSave, onError]);

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
              <Tooltip content="Edit">
                <EditOutlined onClick={editNotes} />
              </Tooltip>
              {extra}
            </Space>
          )
        )
      }
      headStyle={{ marginTop: '16px', minHeight: 'fit-content', paddingInline: '16px' }}
      title={
        <Input
          defaultValue={title}
          disabled={disableTitle || disabled}
          value={editedTitle}
          onBlur={(e) => {
            const newValue = e.currentTarget.value;
            onSaveTitle?.(newValue);
          }}
          onChange={(e) => {
            const newValue = e.currentTarget.value;
            setEditedTitle(newValue);
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

export default NoteCard;
