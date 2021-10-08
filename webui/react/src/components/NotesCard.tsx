import { EditOutlined } from '@ant-design/icons';
import { Button, Card, Layout, Space, Tooltip } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import Markdown from './Markdown';
import css from './NotesCard.module.scss';

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
      bodyStyle={{
        flexGrow: 1,
        flexShrink: 1,
        overflow: 'auto',
        padding: 0,
      }}
      className={css.base}
      extra={(editingNotes ? (
        <Space size="small">
          <Button size="small" onClick={cancelEdit}>Cancel</Button>
          <Button size="small" type="primary" onClick={saveNotes}>Save</Button>
        </Space>
      ) : (
        <Tooltip title="Edit">
          <EditOutlined onClick={editNotes} />
        </Tooltip>

      )
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
        editing={editingNotes}
        markdown={editingNotes ? editedNotes : notes}
        onChange={setEditedNotes} />
    </Card>
  );
};

export default NotesCard;
