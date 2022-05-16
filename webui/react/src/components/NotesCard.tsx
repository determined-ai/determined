import { EditOutlined } from '@ant-design/icons';
import { Button, Card, Space, Tooltip } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Prompt, useLocation } from 'react-router-dom';

import handleError, { ErrorType } from 'utils/error';

import InlineEditor from './InlineEditor';
import Markdown from './Markdown';
import css from './NotesCard.module.scss';
import Spinner from './Spinner';

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

const NotesCard: React.FC<Props> = (
  {
    disabled = false, notes, onSave, onSaveTitle,
    style, title = 'Notes', extra, onChange, noteChangeSignal,
  }: Props,
) => {
  const [ isEditing, setIsEditing ] = useState(false);
  const [ isLoading, setIsLoading ] = useState(false);
  const [ editedNotes, setEditedNotes ] = useState(notes);
  const location = useLocation();

  const existingNotes = useRef(notes);

  useEffect(() => {
    existingNotes.current = notes;
  }, [ notes ]);

  useEffect(() => {
    setIsEditing(false);
    setIsLoading(false);
    setEditedNotes(existingNotes.current);
    // titleRef.current.focus();
  }, [ noteChangeSignal ]);

  const editNotes = useCallback(() => {
    if (disabled) return;
    setIsEditing(true);
  }, [ disabled ]);

  const cancelEdit = useCallback(() => {
    setIsEditing(false);
    setEditedNotes(notes);
  }, [ notes ]);

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
  }, [ editedNotes, onSave ]);

  const handleEditedNotes = useCallback((newNotes: string) => {
    setEditedNotes(newNotes);
    onChange?.(newNotes);
  }, [ onChange ]);

  useEffect(() => {
    setEditedNotes(notes);
    setIsEditing(false);
  }, [ notes ]);

  return (
    <Card
      bodyStyle={{
        flexGrow: 1,
        flexShrink: 1,
        overflow: 'auto',
        padding: 0,
      }}
      className={css.base}
      extra={isEditing ? (
        <Space size="small">
          <Button size="small" onClick={cancelEdit}>Cancel</Button>
          <Button size="small" type="primary" onClick={saveNotes}>Save</Button>
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
      )}
      headStyle={{ minHeight: 'fit-content', paddingInline: 'var(--theme-sizes-layout-big)' }}
      style={{ height: isEditing ? '500px' : '100%', ...style }}
      title={(
        <InlineEditor
          disabled={!onSaveTitle || disabled}
          focusSignal={noteChangeSignal}
          style={{ paddingLeft: '5px', paddingRight: '5px' }}
          value={title}
          onSave={onSaveTitle}
        />
      )}>
      <Spinner spinning={isLoading}>
        <Markdown
          editing={isEditing}
          markdown={isEditing ? editedNotes : notes}
          onChange={handleEditedNotes}
          onClick={(e: React.MouseEvent) => { if (e.detail > 1 || notes === '') editNotes(); }}
        />
      </Spinner>
      <Prompt
        message={(newLocation) => {
          return (
            newLocation.pathname.startsWith(location.pathname.split('/').slice(0, -1).join('/')) ?
              true :
              'You have unsaved notes, are you sure you want to leave? Unsaved notes will be lost.'
          );
        }}
        when={editedNotes !== notes}
      />
    </Card>
  );
};

export default NotesCard;
