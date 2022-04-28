import { Button, Dropdown, Menu } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import { Note } from 'types';

import Icon from './Icon';
import NotesCard from './NotesCard';
import css from './PaginatedNotesCard.module.scss';

interface Props {
  notes: Note[];
  onNewPage: () => void;
  onSave: (notes: Note[]) => void;
}

const PaginatedNotesCard: React.FC<Props> = ({ notes, onNewPage, onSave }:Props) => {
  const [ currentPage, setCurrentPage ] = useState(0);
  const [ editedContents, setEditedContents ] = useState('');
  const [ editedName, setEditedName ] = useState('');
  const [ isEditing, setIsEditing ] = useState(false);

  const handleSwitchPage = useCallback((pageNumber: number) => {
    setCurrentPage(pageNumber);
  }, []);

  const handleNewPage = useCallback(() => {
    const currentPages = notes.length;
    onNewPage();
    setCurrentPage(currentPages);
  }, [ notes.length, onNewPage ]);

  const handleSave = useCallback(async () => {
    await onSave(notes.map((note, idx) => {
      if (idx === currentPage) {
        return { contents: editedContents, name: editedName } as Note;
      }
      return note;
    }));
  }, [ currentPage, editedContents, editedName, notes, onSave ]);

  const handleSaveTitle = useCallback(async (newName: string) => {
    await setEditedName(newName);
  }, []);

  const handleDeletePage = useCallback((pageNumber: number) => {
    onSave(notes.filter((_, idx) => idx === pageNumber));
    if (pageNumber === currentPage){
      setCurrentPage(Math.max(currentPage - 1, 0));
    }
  }, [ currentPage, notes, onSave ]);

  const handleEditNote = useCallback((pageNumber: number) => {
    setCurrentPage(pageNumber);
    setIsEditing(true);
  }, []);

  useEffect(() => {
    if (notes.length === 0) return;
    setEditedContents(notes[currentPage].contents);
    setEditedName(notes[currentPage].name);
  }, [ currentPage, notes ]);

  const ActionMenu = useCallback((pageNumber: number) => {
    return (
      <Menu>
        <Menu.Item onClick={() => handleEditNote(pageNumber)}>Edit</Menu.Item>
        <Menu.Item danger onClick={() => handleDeletePage(pageNumber)}>Delete...</Menu.Item>
      </Menu>
    );
  }, [ handleDeletePage, handleEditNote ]);

  if (notes.length === 0) {
    return (
      <div className={css.emptyBase}>
        <Icon name="document" size="mega" />
        <p>No notes for this project</p>
        <Button onClick={handleNewPage}>+ New Page</Button>
      </div>
    );
  }

  return (
    <div className={css.base}>
      {notes.length > 1 && (
        <div className={css.sidebar}>
          <ul className={css.listContainer} role="list">
            {(notes as Note[]).map((note, idx) => (
              <div className={css.listItemRow} key={idx}>
                <li
                  className={css.listItem}

                  onClick={() => handleSwitchPage(idx)}>
                  {note.name}
                </li>
                <Dropdown
                  className={css.action}
                  overlay={() => ActionMenu(idx)}
                  trigger={[ 'click' ]}>
                  <Icon name="overflow-horizontal" size="big" />
                </Dropdown>
              </div>
            ))}
          </ul>
        </div>
      )}
      <div className={css.notesContainer}>
        <NotesCard
          notes={editedContents}
          startEditing={isEditing}
          title={editedName}
          onSave={handleSave}
          onSaveTitle={handleSaveTitle}
        />
      </div>
    </div>
  );
};

export default PaginatedNotesCard;
