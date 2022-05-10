import { EditOutlined } from '@ant-design/icons';
import { Button, Card, Space, Tooltip } from 'antd';
import React, { useCallback, useState } from 'react';
import { Prompt, useLocation } from 'react-router-dom';

import handleError from 'utils/error';

import { ErrorType } from '../shared/utils/error';

import Markdown from './Markdown';
import css from './NotesCard.module.scss';
import Spinner from './Spinner';

interface Props {
  disabled?: boolean;
  notes: string;
  onSave?: (editedNotes: string) => Promise<void>;
  style?: React.CSSProperties;
}

const NotesCard: React.FC<Props> = ({ disabled = false, notes, onSave, style }: Props) => {
  const [ isEditing, setIsEditing ] = useState(false);
  const [ isLoading, setIsLoading ] = useState(false);
  const [ editedNotes, setEditedNotes ] = useState(notes);
  const location = useLocation();

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
      await onSave?.(editedNotes);
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
          <Tooltip title="Edit">
            <EditOutlined onClick={editNotes} />
          </Tooltip>
        )
      )}
      headStyle={{ paddingInline: 'var(--theme-sizes-layout-big)' }}
      style={{ height: isEditing ? '500px' : '100%', ...style }}
      title="Notes">
      <Spinner spinning={isLoading}>
        <Markdown
          editing={isEditing}
          markdown={isEditing ? editedNotes : notes}
          onChange={setEditedNotes}
          onClick={() => { if (notes === '') editNotes(); }}
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
