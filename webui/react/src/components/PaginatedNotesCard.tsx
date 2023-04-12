import { CheckOutlined } from '@ant-design/icons';
import type { DropDownProps, MenuProps } from 'antd';
import { Dropdown, Modal } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import Button from 'components/kit/Button';
import Empty from 'components/kit/Empty';
import Icon from 'components/kit/Icon';
import Select, { Option, SelectValue } from 'components/kit/Select';
import usePrevious from 'shared/hooks/usePrevious';
import { Note } from 'types';

import NotesCard from './NotesCard';
import css from './PaginatedNotesCard.module.scss';

interface Props {
  disabled?: boolean;
  notes: Note[];
  onDelete: (pageNumber: number) => void;
  onNewPage: () => void;
  onSave: (notes: Note[]) => Promise<void>;
}

const PaginatedNotesCard: React.FC<Props> = ({
  notes,
  onNewPage,
  onSave,
  onDelete,
  disabled = false,
}: Props) => {
  const [currentPage, setCurrentPage] = useState(0);
  const [deleteTarget, setDeleteTarget] = useState(0);
  const [editedContents, setEditedContents] = useState(notes?.[currentPage]?.contents ?? '');
  const [editedName, setEditedName] = useState(notes?.[currentPage]?.name ?? '');
  const [modal, contextHolder] = Modal.useModal();
  const [noteChangeSignal, setNoteChangeSignal] = useState(1);
  const fireNoteChangeSignal = useCallback(
    () => setNoteChangeSignal((prev) => (prev === 100 ? 1 : prev + 1)),
    [setNoteChangeSignal],
  );

  const previousNumberOfNotes = usePrevious(notes.length, undefined);

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
    async (editedNotes: string) => {
      setEditedContents(editedNotes);
      await onSave(
        notes.map((note, idx) => {
          if (idx === currentPage) {
            return { contents: editedNotes, name: editedName } as Note;
          }
          return note;
        }),
      );
    },
    [currentPage, editedName, notes, onSave],
  );

  const handleSaveTitle = useCallback(
    async (newName: string) => {
      setEditedName(newName);
      await onSave(
        notes.map((note, idx) => {
          if (idx === currentPage) {
            return { contents: editedContents ?? note?.contents, name: newName } as Note;
          }
          return note;
        }),
      );
    },
    [currentPage, notes, onSave, editedContents],
  );

  const handleDeletePage = useCallback(
    (deletePageNumber: number) => {
      onDelete(deletePageNumber);
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
    setEditedName((prev) => notes?.[currentPage]?.name ?? prev);
  }, [currentPage, notes]);

  const ActionMenu = useCallback(
    (pageNumber: number): DropDownProps['menu'] => {
      const onItemClick: MenuProps['onClick'] = (e) => {
        e.domEvent.stopPropagation();
        handleDeletePage(pageNumber);
      };
      const menuItems: MenuProps['items'] = [{ danger: true, key: 'delete', label: 'Delete...' }];
      return { items: menuItems, onClick: onItemClick };
    },
    [handleDeletePage],
  );

  if (notes.length === 0) {
    return (
      <Empty
        description={
          <>
            <p>No notes for this project</p>
            <Button onClick={handleNewPage}>+ New Page</Button>
          </>
        }
        icon="document"
      />
    );
  }

  return (
    <div className={css.base}>
      {notes.length > 0 && (
        <div className={css.sidebar}>
          <ul className={css.listContainer} role="list">
            {(notes as Note[]).map((note, idx) => (
              <Dropdown
                disabled={disabled}
                key={idx}
                menu={ActionMenu(idx)}
                trigger={['contextMenu']}>
                <li
                  className={css.listItem}
                  style={{
                    borderColor:
                      idx === currentPage ? 'var(--theme-stage-border-strong)' : undefined,
                  }}
                  onClick={() => handleSwitchPage(idx)}>
                  <span>{note.name}</span>
                  {!disabled && (
                    <Dropdown menu={ActionMenu(idx)} trigger={['click']}>
                      <div className={css.action} onClick={(e) => e.stopPropagation()}>
                        <Icon name="overflow-horizontal" />
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
              <Option className={css.selectOption} key={idx} value={idx}>
                <CheckOutlined
                  className={css.currentPage}
                  style={{
                    marginRight: 8,
                    visibility: idx === currentPage ? 'visible' : 'hidden',
                  }}
                />
                <span>{note.name}</span>
              </Option>
            );
          })}
        </Select>
      </div>
      <div className={css.notesContainer}>
        <NotesCard
          disabled={disabled}
          extra={
            <Dropdown menu={ActionMenu(currentPage)} trigger={['click']}>
              <div style={{ cursor: 'pointer' }}>
                <Icon name="overflow-horizontal" />
              </div>
            </Dropdown>
          }
          noteChangeSignal={noteChangeSignal}
          notes={notes?.[currentPage]?.contents ?? ''}
          style={{ border: 0 }}
          title={notes?.[currentPage]?.name ?? ''}
          onChange={handleEditedNotes}
          onSave={handleSave}
          onSaveTitle={handleSaveTitle}
        />
      </div>
      {contextHolder}
    </div>
  );
};

export default PaginatedNotesCard;
