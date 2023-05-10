import React, { useState } from 'react';

import NoteCard, { Props as NoteProps } from 'components/kit/NoteCard';
import NoteCards, { Props as NotesProps } from 'components/kit/NoteCards';
import { Note } from 'types';

export const useNote = (): ((props?: NoteProps) => JSX.Element) => {
  const [note, setNote] = useState('');
  const [title, setTitle] = useState('Untitled');
  const onSave = async (n: string) => await setNote(n);
  const onSaveTitle = async (n: string) => await setTitle(n);
  return (props) => (
    <NoteCard notes={note} title={title} onSave={onSave} onSaveTitle={onSaveTitle} {...props} />
  );
};

export const useNotes = (): ((props?: NotesProps) => JSX.Element) => {
  const [notes, setNotes] = useState<Note[]>([]);
  const onDelete = (p: number) =>
    setNotes((n) => {
      n.splice(p, 1);
      return [...n];
    });
  const onNewPage = () =>
    setNotes((n) => {
      n.push({ contents: '', name: 'Untitled' });
      return [...n];
    });
  const onSave = async (n: Note[]) => await setNotes(n);
  return (props) => (
    <NoteCards notes={notes} onDelete={onDelete} onNewPage={onNewPage} onSave={onSave} {...props} />
  );
};
