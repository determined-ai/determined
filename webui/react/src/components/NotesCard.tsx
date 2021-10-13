import { EditOutlined } from '@ant-design/icons';
import { Button, Card, Space, Tooltip } from 'antd';
import React, { useCallback, useState } from 'react';
import { Prompt, useLocation } from 'react-router-dom';

import Markdown from './Markdown';
import css from './NotesCard.module.scss';

interface Props {
  notes: string;
  onSave: (editedNotes: string) => void;
  style?: React.CSSProperties;
}

const NotesCard: React.FC<Props> = ({ notes, onSave, style }: Props) => {
  const [ isEditing, setIsEditing ] = useState(false);
  const [ editedNotes, setEditedNotes ] = useState(notes);
  const location = useLocation();

  const editNotes = useCallback(() => {
    setIsEditing(true);
  }, []);

  const cancelEdit = useCallback(() => {
    setIsEditing(false);
    setEditedNotes(notes);
  }, [ notes ]);

  const saveNotes = useCallback(() => {
    setIsEditing(false);
    onSave(editedNotes);
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
        <Tooltip title="Edit">
          <EditOutlined onClick={editNotes} />
        </Tooltip>
      )}
      headStyle={{
        flexGrow: 0,
        flexShrink: 0,
        paddingLeft: 'var(--theme-sizes-layout-big)',
        paddingRight: 'var(--theme-sizes-layout-big)',
      }}
      style={style}
      title="Notes">
      <Markdown
        editing={isEditing}
        markdown={isEditing ? editedNotes : notes}
        onChange={setEditedNotes}
        onClick={() => { if (notes === '') editNotes(); }}
      />
      <Prompt
        message={(newLocation) => {
          return (
            newLocation.pathname.startsWith(location.pathname.split('/').slice(0, -1).join('/')) ?
              true :
              'You have unsaved notes, are you sure you want to leave? Unsaved notes will be lost.'
          );
        }}
        when={editedNotes !== notes} />
    </Card>
  );
};

export default NotesCard;
