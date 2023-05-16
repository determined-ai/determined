import React, { useState } from 'react';

import Notes, { Props } from 'components/kit/Notes';
import { Note } from 'types';

export const useNoteDemo = (): ((props?: Omit<Props, 'multiple'>) => JSX.Element) => {
  const [note, setNote] = useState<Note>({ contents: '', name: 'Untitled' });
  const onSave = async (n: Note) => await setNote(n);
  return (props) => <Notes {...props} notes={note} onSave={onSave} />;
};

export const useNotesDemo = (): ((props?: Props) => JSX.Element) => {
  const [notes, setNotes] = useState<Note[]>([]);
  const onDelete = (p: number) => setNotes((n) => n.filter((_, idx) => idx !== p));
  const onNewPage = () => setNotes((n) => [...n, { contents: '', name: 'Untitled' }]);
  const onSave = async (n: Note[]) => await setNotes(n);
  return (props) => (
    <Notes
      {...props}
      multiple
      notes={notes}
      onDelete={onDelete}
      onNewPage={onNewPage}
      onSave={onSave}
    />
  );
};
