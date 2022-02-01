import { Button, Input, Modal, ModalFuncProps } from 'antd';
import { ModalFunc } from 'antd/es/modal/confirm';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FixedSizeList as List } from 'react-window';

import { isEqual } from 'utils/data';
import { camelCaseToSentence } from 'utils/string';

import css from './useCustomizeColumnsModal.module.scss';

interface ModalState {
  columns: string[];
  defaultVisibleColumns: string[];
  initialVisibleColumns: string[];
  onSave?: (columns: string[]) => void;
  visible: boolean;
}

export interface ShowModalProps {
  columns: string[];
  defaultVisibleColumns: string[];
  initialVisibleColumns: string[];
  onSave?: (columns: string[]) => void;
}

interface ModalHooks {
  showModal: (props: ShowModalProps) => void;
}

const useCustomizeColumnsModal = (): ModalHooks => {
  const modalRef = useRef<ReturnType<ModalFunc>>();
  const [ modalState, setModalState ] = useState<ModalState>({
    columns: [],
    defaultVisibleColumns: [],
    initialVisibleColumns: [],
    visible: false,
  });
  const [ searchTerm, setSearchTerm ] = useState('');
  const [ visibleColumns, setVisibleColumns ] = useState<string[]>([]);

  const showModal = useCallback((
    { columns, initialVisibleColumns, defaultVisibleColumns, onSave }: ShowModalProps,
  ) => {
    setModalState({ columns, defaultVisibleColumns, initialVisibleColumns, onSave, visible: true });
    setVisibleColumns(initialVisibleColumns);
  }, []);

  const closeModal = useCallback(() => {
    if (!modalRef.current) return;
    modalRef.current.destroy();
    modalRef.current = undefined;
  }, []);

  const handleCancel = useCallback(() => {
    if (!modalRef.current) return;
    closeModal();
  }, [ closeModal ]);

  const handleSave = useCallback((state: ModalState) => {
    if (!modalRef.current) return;
    state.onSave?.(visibleColumns);
    closeModal();
  }, [ closeModal, visibleColumns ]);

  const handleSearch = useCallback((e) => {
    setSearchTerm(e.target.value.toLowerCase());
  }, []);

  const hiddenColumns = useMemo(() => {
    return modalState.columns.filter(column => !visibleColumns.includes(column));
  }, [ modalState.columns, visibleColumns ]);

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
      setVisibleColumns(prev => prev.filter(column => !transfer.includes(column)));
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

  const resetColumns = useCallback((state: ModalState) => {
    setVisibleColumns(state.defaultVisibleColumns);
  }, []);

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

  const generateModalContent = useCallback((state: ModalState): React.ReactNode => {
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
              {!isEqual(state.defaultVisibleColumns, visibleColumns) && (
                <Button type="link" onClick={() => resetColumns(state)}>
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
  }, [ handleSearch,
    filteredHiddenColumns,
    renderHiddenRow,
    visibleColumns,
    filteredVisibleColumns,
    renderVisibleRow,
    makeVisible,
    resetColumns,
    makeHidden ]);

  const generateModalProps = useCallback((state: ModalState): Partial<ModalFuncProps> => {
    const modalProps: Partial<ModalFuncProps> = {
      bodyStyle: { padding: 0 },
      className: css.base,
      closable: true,
      content: generateModalContent(state),
      icon: null,
      maskClosable: true,
      okText: 'Save',
      onCancel: handleCancel,
      onOk: () => handleSave(state),
      title: 'Customize Columns',
    };

    return modalProps;
  }, [ generateModalContent, handleCancel, handleSave ]);

  // Detect modal state change and update.
  useEffect(() => {
    if (!modalState.visible) return;

    const modalProps = generateModalProps(modalState);
    if (modalRef.current) {
      modalRef.current.update(prev => ({ ...prev, ...modalProps }));
    } else {
      modalRef.current = Modal.confirm(modalProps);
    }
  }, [ generateModalProps, modalState ]);

  // When the component using the hook unmounts, remove the modal automatically.
  useEffect(() => {
    return () => {
      if (!modalRef.current) return;
      modalRef.current.destroy();
      modalRef.current = undefined;
    };
  }, []);

  return { showModal };
};

export default useCustomizeColumnsModal;
