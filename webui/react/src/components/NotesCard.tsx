import { EditOutlined } from '@ant-design/icons';
import { Button, Card, Tooltip } from 'antd';
import React, { useCallback, useState } from 'react';

import Markdown from './Markdown';

interface Props {
  notes: string;
  onSave: (editedNotes: string) => void;
  style?: React.CSSProperties;
}

const NotesCard: React.FC<Props> = ({ notes, onSave, style }: Props) => {
  const [ editingNotes, setEditingNotes ] = useState(false);
  const [ editedNotes, setEditedNotes ] = useState(notes);

  const editNotes = useCallback(() => {
    setEditingNotes(true);
  }, []);

  const cancelEdit = useCallback(() => {
    setEditingNotes(false);
    setEditedNotes(notes);
  }, [ notes ]);

  const saveNotes = useCallback(() => {
    setEditingNotes(false);
    onSave(editedNotes);
  }, [ editedNotes, onSave ]);

  return (
    <Card
      bodyStyle={{ paddingTop: 'var(--theme-sizes-layout-large)' }}
      extra={(editingNotes ?
        <div style={{ display: 'flex', gap: 4 }}>
          <Button onClick={cancelEdit}>Cancel</Button>
          <Button onClick={saveNotes}>Save</Button>
        </div> :
        <Tooltip title="Edit">
          <EditOutlined onClick={editNotes} />
        </Tooltip>
      )}
      style={style}
      title="Notes">
      <Markdown
        editing={editingNotes}
        height={500}
        markdown={editingNotes ? editedNotes : notes}
        onChange={setEditedNotes} />
    </Card>
  );
};

export default NotesCard;
