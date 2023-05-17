import { Note } from 'types';

import NoteCard from './NoteCard';
import NoteCards from './NoteCards';

export type Props =
  | {
      multiple: true;
      disabled?: boolean;
      disableTitle?: boolean;
      notes: Note[];
      onDelete?: (pageNumber: number) => void;
      onNewPage: () => void;
      onSave: (notes: Note[]) => Promise<void>;
    }
  | {
      multiple?: false;
      disabled?: boolean;
      disableTitle?: boolean;
      notes: Note;
      onSave: (notes: Note) => Promise<void>;
    };

const Notes: React.FC<Props> = ({
  multiple,
  notes,
  onSave,
  disabled = false,
  disableTitle,
  ...props
}: Props) => {
  return multiple ? (
    <NoteCards
      disabled={disabled}
      notes={notes}
      onDelete={'onDelete' in props ? props.onDelete : undefined}
      // eslint-disable-next-line @typescript-eslint/no-empty-function
      onNewPage={'onNewPage' in props ? props.onNewPage : () => {}}
      onSave={onSave}
    />
  ) : (
    <NoteCard disabled={disabled} disableTitle={disableTitle} note={notes} onSaveNote={onSave} />
  );
};

export default Notes;
