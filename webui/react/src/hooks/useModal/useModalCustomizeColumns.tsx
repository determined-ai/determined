import { Button, Input, ModalFuncProps } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { FixedSizeList as List } from 'react-window';

import { isEqual } from 'utils/data';
import { camelCaseToSentence } from 'utils/string';

import useModal, { ModalHooks as Hooks } from './useModal';
import css from './useModalCustomizeColumns.module.scss';

interface Props {
  columns: string[];
  defaultVisibleColumns: string[];
  onSave?: (columns: string[]) => void;
}

export interface ShowModalProps {
  initialModalProps?: ModalFuncProps;
  initialVisibleColumns?: string[];
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (props: ShowModalProps) => void;
}

const useModalCustomizeColumns = ({
  columns,
  defaultVisibleColumns,
  onSave,
}: Props): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal();
  const [ columnList ] = useState(columns); //this is only to prevent rerendering
  const [ searchTerm, setSearchTerm ] = useState('');
  const [ visibleColumns, setVisibleColumns ] = useState<string[]>([]);

  const handleSave = useCallback(() => {
    onSave?.(visibleColumns);
  }, [ onSave, visibleColumns ]);

  const handleSearch = useCallback((e) => {
    setSearchTerm(e.target.value);
  }, []);

  const hiddenColumns = useMemo(() => {
    const visibleColumnsSet = new Set(visibleColumns);
    return columnList.filter((column) => !visibleColumnsSet.has(column));
  }, [ columnList, visibleColumns ]);

  const filteredHiddenColumns = useMemo(() => {
    const regex = RegExp(searchTerm, 'i');
    return hiddenColumns.filter(column => regex.test(camelCaseToSentence(column)));
  }, [ hiddenColumns, searchTerm ]);

  const filteredVisibleColumns = useMemo(() => {
    const regex = RegExp(searchTerm, 'i');
    return visibleColumns.filter(column => regex.test(camelCaseToSentence(column)));
  }, [ visibleColumns, searchTerm ]);

  const makeHidden = useCallback((transfer: string | string[]) => {
    if (Array.isArray(transfer)) {
      const transferSet = new Set(transfer);
      setVisibleColumns(prev => prev.filter(column => !transferSet.has(column)));
    } else {
      setVisibleColumns(prev => prev.filter(column => transfer !== column));
    }
  }, []);

  const makeVisible = useCallback((transfer: string | string[]) => {
    if (Array.isArray(transfer)) {
      setVisibleColumns(prev => [ ...prev, ...transfer ]);
    } else {
      setVisibleColumns(prev => [ ...prev, transfer ]);
    }
  }, []);

  const resetColumns = useCallback(() => {
    setVisibleColumns(defaultVisibleColumns);
  }, [ defaultVisibleColumns ]);

  const renderColumnName = useCallback((columnName: string) => {
    const sentenceColumnName = columnName === 'id' ? 'ID' : camelCaseToSentence(columnName);
    const regex = new RegExp(searchTerm, 'i');
    if (searchTerm === '' || !regex.test(sentenceColumnName)){
      return <span>{sentenceColumnName}</span>;
    }
    const searchIndex = sentenceColumnName.search(regex);
    return (
      <span>{sentenceColumnName.slice(0, searchIndex)}
        <mark>{sentenceColumnName.match(regex)?.[0]}</mark>
        {sentenceColumnName.slice(searchIndex + searchTerm.length)}
      </span>
    );
  }, [ searchTerm ]);

  const renderRow = useCallback((row, style, handleClick) => {
    return (
      <li style={style} onClick={handleClick}>
        {renderColumnName(row)}
      </li>
    );
  }, [ renderColumnName ]);

  const renderHiddenRow = useCallback(({ index, style }) => {
    const row = filteredHiddenColumns[index];
    return renderRow(row, style, () => makeVisible(row));
  }, [ filteredHiddenColumns, makeVisible, renderRow ]);

  const renderVisibleRow = useCallback(({ index, style }) => {
    const row = filteredVisibleColumns[index];
    return renderRow(row, style, () => makeHidden(row));
  }, [ filteredVisibleColumns, makeHidden, renderRow ]);

  const modalContent = useMemo((): React.ReactNode => {
    // We always render the form regardless of mode to provide a reference to it.
    return (
      <div className={css.base}>
        <Input placeholder="Search columns..." onChange={handleSearch} />
        <div className={css.columns}>
          <div className={css.column}>
            <h2>Hidden</h2>
            <List
              className={css.listContainer}
              height={200}
              innerElementType="ul"
              itemCount={filteredHiddenColumns.length}
              itemSize={24}
              width="100%">
              {renderHiddenRow}
            </List>
            <Button type="link" onClick={() => makeVisible(filteredHiddenColumns)}>
              Add All
            </Button>
          </div>
          <div className={css.column}>
            <div className={css.visibleTitleRow}>
              <h2>Visible</h2>
              {!isEqual(defaultVisibleColumns, visibleColumns) && (
                <Button type="link" onClick={resetColumns}>
                  Reset
                </Button>
              )}
            </div>
            <List
              className={css.listContainer}
              height={200}
              innerElementType="ul"
              itemCount={filteredVisibleColumns.length}
              itemSize={24}
              width="100%">
              {renderVisibleRow}
            </List>
            <Button type="link" onClick={() => makeHidden(filteredVisibleColumns)}>
              Remove All
            </Button>
          </div>
        </div>
      </div>
    );
  }, [ defaultVisibleColumns,
    handleSearch,
    filteredHiddenColumns,
    renderHiddenRow,
    visibleColumns,
    filteredVisibleColumns,
    renderVisibleRow,
    makeVisible,
    resetColumns,
    makeHidden ]);

  const modalProps: Partial<ModalFuncProps> = useMemo(() => {
    return {
      bodyStyle: { padding: 0 },
      className: css.base,
      closable: true,
      content: modalContent,
      icon: null,
      maskClosable: true,
      okText: 'Save',
      onOk: handleSave,
      title: 'Customize Columns',
    };
  }, [ modalContent, handleSave ]);

  const modalOpen = useCallback(({ initialVisibleColumns, initialModalProps }: ShowModalProps) => {
    setVisibleColumns(initialVisibleColumns ?? defaultVisibleColumns);
    openOrUpdate({ ...modalProps, ...initialModalProps });
  }, [ defaultVisibleColumns, modalProps, openOrUpdate ]);

  /*
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(modalProps);
  }, [ modalProps, modalRef, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalCustomizeColumns;
