import { ErrorHandler, Note } from 'components/kit/internal/types';

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
      onError: ErrorHandler;
    }
  | {
      multiple?: false;
      disabled?: boolean;
      disableTitle?: boolean;
      notes: Note;
      onSave: (notes: Note) => Promise<void>;
      onError: ErrorHandler;
    };

const Notes: React.FC<Props> = ({
  multiple,
  notes,
  onError,
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
      onError={onError}
      // eslint-disable-next-line @typescript-eslint/no-empty-function
      onNewPage={'onNewPage' in props ? props.onNewPage : () => {}}
      onSave={onSave}
    />
  ) : (
    <NoteCard
      disabled={disabled}
      disableTitle={disableTitle}
      note={notes}
      onError={onError}
      onSaveNote={onSave}
    />
  );
};

export default Notes;
