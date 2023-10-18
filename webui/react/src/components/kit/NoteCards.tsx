import { Modal } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Button from 'components/kit/Button';
import Dropdown from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import { ErrorHandler, Note } from 'components/kit/internal/types';
import usePrevious from 'components/kit/internal/usePrevious';
import Message from 'components/kit/Message';
import Select, { Option, SelectValue } from 'components/kit/Select';

import NoteCard from './NoteCard';
import css from './NoteCards.module.scss';

interface Props {
  disabled?: boolean;
  notes: Note[];
  onError: ErrorHandler;
  onDelete?: (pageNumber: number) => void;
  onNewPage: () => void;
  onSave: (notes: Note[]) => Promise<void>;
}

const NoteCards: React.FC<Props> = ({
  notes,
  onNewPage,
  onSave,
  onDelete,
  onError,
  disabled = false,
}: Props) => {
  const [currentPage, setCurrentPage] = useState(0);
  const [deleteTarget, setDeleteTarget] = useState(0);
  const [editedContents, setEditedContents] = useState(notes?.[currentPage]?.contents ?? '');
  const [modal, contextHolder] = Modal.useModal();
  const [noteChangeSignal, setNoteChangeSignal] = useState(1);
  const fireNoteChangeSignal = useCallback(
    () => setNoteChangeSignal((prev) => (prev === 100 ? 1 : prev + 1)),
    [setNoteChangeSignal],
  );

  const previousNumberOfNotes = usePrevious(notes.length, undefined);

  const DROPDOWN_MENU = useMemo(
    () => [{ danger: true, disabled: !onDelete, key: 'delete', label: 'Delete...' }],
    [onDelete],
  );

  const handleSwitchPage = useCallback(
    (pageNumber: number | SelectValue) => {
      if (pageNumber === currentPage) return;
      if (editedContents !== notes?.[currentPage]?.contents) {
        modal.confirm({
          content: (
            <p>
              You have unsaved notes, are you sure you want to switch pages? Unsaved notes will be
              lost.
            </p>
          ),
          onOk: () => {
            setCurrentPage(pageNumber as number);
            fireNoteChangeSignal();
          },
          title: 'Unsaved content',
        });
      } else {
        setCurrentPage(pageNumber as number);
        setEditedContents(notes?.[currentPage]?.contents ?? '');
        fireNoteChangeSignal();
      }
    },
    [currentPage, editedContents, modal, notes, fireNoteChangeSignal],
  );

  useEffect(() => {
    if (previousNumberOfNotes == null) {
      if (notes.length) {
        handleSwitchPage(0);
        fireNoteChangeSignal();
      }
    } else if (notes.length > previousNumberOfNotes) {
      handleSwitchPage(notes.length - 1);
    } else if (notes.length < previousNumberOfNotes) {
      // dont call handler here because page isn't actually switching
      setCurrentPage((prevPageNumber) =>
        prevPageNumber > deleteTarget ? prevPageNumber - 1 : prevPageNumber,
      );
    }
  }, [previousNumberOfNotes, notes.length, deleteTarget, handleSwitchPage, fireNoteChangeSignal]);

  const handleNewPage = useCallback(() => {
    const currentPages = notes.length;
    onNewPage();
    handleSwitchPage(currentPages);
  }, [notes.length, onNewPage, handleSwitchPage]);

  const handleSave = useCallback(
    async (note: Note) => {
      setEditedContents(note.contents);
      await onSave(notes.map((n, idx) => (idx === currentPage ? note : n)));
    },
    [currentPage, notes, onSave],
  );

  const handleDeletePage = useCallback(
    (deletePageNumber: number) => {
      onDelete?.(deletePageNumber);
      setDeleteTarget(deletePageNumber);
    },
    [onDelete, setDeleteTarget],
  );

  const handleEditedNotes = useCallback((newContents: string) => {
    setEditedContents(newContents);
  }, []);

  useEffect(() => {
    if (notes.length === 0) return;
    if (currentPage < 0) {
      setCurrentPage(0);
      fireNoteChangeSignal();
    }
    if (currentPage >= notes.length) {
      setCurrentPage(notes.length - 1);
      fireNoteChangeSignal();
    }
  }, [currentPage, notes.length, fireNoteChangeSignal]);

  useEffect(() => {
    setEditedContents((prev) => notes?.[currentPage]?.contents ?? prev);
  }, [currentPage, notes]);

  const handleDropdown = useCallback(
    (pageNumber: number) => handleDeletePage(pageNumber),
    [handleDeletePage],
  );

  if (notes.length === 0) {
    return (
      <Message
        description={
          <>
            <p>No notes for this project</p>
            {!disabled && <Button onClick={handleNewPage}>+ New Note</Button>}
          </>
        }
        icon="document"
      />
    );
  }

  return (
    <>
      <div className={css.tabOptions}>
        {!disabled && (
          <Button type="text" onClick={onNewPage}>
            + New Note
          </Button>
        )}
      </div>
      <div className={css.base}>
        {notes.length > 0 && (
          <div className={css.sidebar}>
            <ul className={css.listContainer} role="list">
              {(notes as Note[]).map((note, idx) => (
                <Dropdown
                  disabled={disabled}
                  isContextMenu
                  key={idx}
                  menu={DROPDOWN_MENU}
                  onClick={() => handleDropdown(idx)}>
                  <li
                    className={css.listItem}
                    style={{
                      borderColor:
                        idx === currentPage ? 'var(--theme-stage-border-strong)' : undefined,
                    }}
                    onClick={() => handleSwitchPage(idx)}>
                    <span>{note.name}</span>
                    {!disabled && (
                      <Dropdown menu={DROPDOWN_MENU} onClick={() => handleDropdown(idx)}>
                        <div className={css.action} onClick={(e) => e.stopPropagation()}>
                          <Icon name="overflow-horizontal" title="Action menu" />
                        </div>
                      </Dropdown>
                    )}
                  </li>
                </Dropdown>
              ))}
            </ul>
          </div>
        )}
        <div className={css.pageSelectRow}>
          <Select value={currentPage} onSelect={handleSwitchPage}>
            {notes.map((note, idx) => {
              return (
                <Option key={idx} value={idx}>
                  <span
                    style={{
                      marginRight: 8,
                      visibility: idx === currentPage ? 'visible' : 'hidden',
                    }}>
                    <Icon decorative name="checkmark" size="small" />
                  </span>
                  <span>{note.name}</span>
                </Option>
              );
            })}
          </Select>
        </div>
        <div className={css.notesContainer}>
          <NoteCard
            disabled={disabled}
            extra={
              <Dropdown menu={DROPDOWN_MENU} onClick={() => handleDropdown(currentPage)}>
                <Button
                  icon={<Icon name="overflow-horizontal" title="Action menu" />}
                  type="text"
                />
              </Dropdown>
            }
            note={notes?.[currentPage]}
            noteChangeSignal={noteChangeSignal}
            onChange={handleEditedNotes}
            onError={onError}
            onSaveNote={handleSave}
          />
        </div>
        {contextHolder}
      </div>
    </>
  );
};

export default NoteCards;
