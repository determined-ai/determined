import { EditOutlined } from '@ant-design/icons';
import { Button, Card, Space, Tooltip } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useLocation } from 'react-router-dom';

import Spinner from 'shared/components/Spinner/Spinner';
import history from 'shared/routes/history';
import { ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

import InlineEditor from './InlineEditor';
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
  const location = useLocation();

  const existingNotes = useRef(notes);

  useEffect(() => {
    existingNotes.current = notes;
  }, [notes]);

  useEffect(() => {
    setIsEditing(false);
    setIsLoading(false);
    setEditedNotes(existingNotes.current);
  }, [noteChangeSignal]);

  useEffect(() => {
    // TODO: This is an alternative of Prompt from react-router-dom
    // As soon as react-router-domv6 supports Prompt, replace this with Promt
    const unblock = isEditing
      ? history.block((tx) => {
          const pathnames = ['notes', 'models', 'projects'];
          let isAllowedNavigation = true;

          // check pathname if one of these names is included
          if (pathnames.some((name) => location.pathname.includes(name)) && notes !== editedNotes) {
            isAllowedNavigation = window.confirm(
              'You have unsaved notes, are you sure you want to leave? Unsaved notes will be lost.',
            );
          }
          if (isAllowedNavigation) {
            unblock();
            tx.retry();
          }
          return isAllowedNavigation;
        })
      : () => undefined;

    return () => unblock();
  }, [editedNotes, isEditing, location.pathname, notes]);

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
      headStyle={{ minHeight: 'fit-content', paddingInline: '16px' }}
      style={{ height: '100%', ...style }}
      title={
        <InlineEditor
          disabled={!onSaveTitle || disabled}
          focusSignal={noteChangeSignal}
          style={{ paddingLeft: '5px', paddingRight: '5px' }}
          value={title}
          onSave={onSaveTitle}
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
