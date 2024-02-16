import Input from 'hew/Input';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Virtuoso } from 'react-virtuoso';

import Link from 'components/Link';

import DraggableListItem from './DraggableListItem';
import css from './Transfer.module.scss';

interface Props {
  defaultTargetEntries: string[];
  entries: string[];
  initialTargetEntries?: string[];
  onChange?: (targetList: string[]) => void;
  placeholder?: string;
  reorder?: boolean;
  sourceListTitle?: string;
  targetListTitle?: string;
  persistentEntries?: string[]; // Entries still exist when clicking "Remove all"
}

const Transfer: React.FC<Props> = ({
  entries,
  defaultTargetEntries,
  initialTargetEntries,
  sourceListTitle = 'Source',
  targetListTitle = 'Target',
  placeholder = 'Search entries...',
  reorder = true,
  persistentEntries,
  onChange,
}: Props) => {
  const [targetEntries, setTargetEntries] = useState<string[]>(
    initialTargetEntries ?? defaultTargetEntries ?? [],
  );
  const [searchTerm, setSearchTerm] = useState('');

  const handleSearch = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(e.target.value);
  }, []);

  const hiddenEntries = useMemo(() => {
    const targetEntriesSet = new Set(targetEntries);
    return entries.filter((entry) => !targetEntriesSet.has(entry));
  }, [entries, targetEntries]);

  const filteredHiddenEntries = useMemo(() => {
    const regex = RegExp(searchTerm, 'i');
    return hiddenEntries.filter((entry) => regex.test(entry));
  }, [hiddenEntries, searchTerm]);

  const filteredVisibleEntries = useMemo(() => {
    const regex = RegExp(searchTerm, 'i');
    return targetEntries.filter((entry) => regex.test(entry));
  }, [targetEntries, searchTerm]);

  const moveToLeft = useCallback((transfer: string | string[]) => {
    if (Array.isArray(transfer)) {
      const transferSet = new Set(transfer);
      setTargetEntries((prev) => prev.filter((entry) => !transferSet.has(entry)));
    } else {
      setTargetEntries((prev) => prev.filter((entry) => transfer !== entry));
    }
  }, []);

  const moveToRight = useCallback((transfer: string | string[]) => {
    if (Array.isArray(transfer)) {
      setTargetEntries((prev) => [...prev, ...transfer]);
    } else {
      setTargetEntries((prev) => [...prev, transfer]);
    }
  }, []);

  const resetEntries = useCallback(() => {
    setTargetEntries(defaultTargetEntries);
  }, [defaultTargetEntries]);

  useEffect(() => {
    onChange?.(targetEntries);
  }, [onChange, targetEntries]);

  const renderEntry = useCallback(
    (entryName: string) => {
      const renameEntry = (): string => {
        switch (entryName) {
          case 'id':
            return 'ID';
          case 'startTime':
            return 'Started';
          case 'searcherType':
            return 'Searcher';
          case 'forkedFrom':
            return 'Forked';
          case 'numTrials':
            return 'Trials';
          default:
            return entryName;
        }
      };
      const sentenceEntryName = renameEntry();
      const regex = new RegExp(searchTerm, 'i');
      if (searchTerm === '' || !regex.test(sentenceEntryName)) {
        return <span>{sentenceEntryName}</span>;
      }
      const searchIndex = sentenceEntryName.search(regex);
      return (
        <span>
          {sentenceEntryName.slice(0, searchIndex)}
          <mark>{sentenceEntryName.match(regex)?.[0]}</mark>
          {sentenceEntryName.slice(searchIndex + searchTerm.length)}
        </span>
      );
    },
    [searchTerm],
  );

  const renderRow = useCallback(
    (row: string, handleClick: () => void) => {
      return <li onClick={handleClick}>{renderEntry(row)}</li>;
    },
    [renderEntry],
  );

  const switchRowOrder = useCallback((entry: string, newNeighborEntry: string) => {
    if (entry !== newNeighborEntry) {
      setTargetEntries((prev) => {
        const updatedVisibleEntries = [...prev];
        const entryIndex = updatedVisibleEntries.findIndex((entryName) => entryName === entry);
        const newNeighborEntryIndex = updatedVisibleEntries.findIndex(
          (entryName) => entryName === newNeighborEntry,
        );
        updatedVisibleEntries.splice(entryIndex, 1);
        updatedVisibleEntries.splice(newNeighborEntryIndex, 0, entry);
        return updatedVisibleEntries;
      });
    }
  }, []);

  const renderDraggableRow = useCallback(
    (
      row: string,
      index: number,
      handleClick: (event: React.MouseEvent<Element, MouseEvent>) => void,
    ) => {
      return (
        <DraggableListItem
          columnName={row}
          index={index}
          onClick={handleClick}
          onDrop={switchRowOrder}>
          {renderEntry(row)}
        </DraggableListItem>
      );
    },
    [renderEntry, switchRowOrder],
  );

  const renderHiddenRow = useCallback(
    (_index: number, row: string) => {
      return renderRow(row, () => moveToRight(row));
    },
    [moveToRight, renderRow],
  );

  const renderVisibleRow = useCallback(
    (index: number, row: string) => {
      return reorder
        ? renderDraggableRow(row, index, () => moveToLeft(row))
        : renderRow(row, () => moveToLeft(row));
    },
    [moveToLeft, renderDraggableRow, renderRow, reorder],
  );

  return (
    <div className={css.base}>
      <Input placeholder={placeholder} onChange={handleSearch} />
      <div className={css.entries}>
        <div className={css.column}>
          <h2>{sourceListTitle}</h2>
          <ul className={css.listContainer}>
            <Virtuoso
              data={filteredHiddenEntries}
              itemContent={(index, data) => renderHiddenRow(index, data)}
              style={{ height: '200px' }}
              totalCount={filteredHiddenEntries.length}
            />
          </ul>
          <Link onClick={() => moveToRight(filteredHiddenEntries)}>Add All</Link>
        </div>
        <div className={css.column}>
          <div className={css.targetTitleRow}>
            <h2>{targetListTitle}</h2>
            {!_.isEqual(defaultTargetEntries, targetEntries) && (
              <Link onClick={resetEntries}>Reset</Link>
            )}
          </div>
          <ul className={css.listContainer}>
            <Virtuoso
              data={filteredVisibleEntries}
              itemContent={(index, data) => renderVisibleRow(index, data)}
              style={{ height: '200px' }}
              totalCount={filteredVisibleEntries.length}
            />
          </ul>
          <Link
            onClick={() => {
              moveToLeft(filteredVisibleEntries);
              // removing everything was keeping the columns out of sync with the UI...
              if (persistentEntries && persistentEntries.length > 0) moveToRight(persistentEntries);
            }}>
            Remove All
          </Link>
        </div>
      </div>
    </div>
  );
};

export default Transfer;
