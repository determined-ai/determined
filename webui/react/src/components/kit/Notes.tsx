import { useCallback } from 'react';

import { Note } from 'types';

import NoteCard from './NoteCard';
import NoteCards from './NoteCards';

export interface Props {
  disabled?: boolean;
  disableTitle?: boolean;
  notes: Note[];
  onDelete?: (pageNumber: number) => void;
  onNewPage?: () => void; // onNewPage is the indicator for single or multiple notes.
  onSave: (notes: Note[]) => Promise<void>;
}

const Notes: React.FC<Props> = ({
  notes,
  onNewPage,
  onSave,
  onDelete,
  disabled = false,
  disableTitle,
}: Props) => {
  const onSaveNote = useCallback(
    async (n: Note) => {
      await onSave([n]);
    },
    [onSave],
  );

  return onNewPage ? (
    <NoteCards
      disabled={disabled}
      notes={notes}
      onDelete={onDelete}
      onNewPage={onNewPage}
      onSave={onSave}
    />
  ) : (
    <NoteCard
      disabled={disabled}
      disableTitle={disableTitle}
      note={notes[0]}
      onSaveNote={onSaveNote}
    />
  );
};

export default Notes;
