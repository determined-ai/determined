import { Button, Input, Modal, ModalFuncProps } from 'antd';
import { ModalFunc } from 'antd/es/modal/confirm';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { camelCaseToSentence, sentenceToCamelCase } from 'utils/string';

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
    return hiddenColumns.filter(column => column.toLowerCase().includes(searchTerm));
  }, [ hiddenColumns, searchTerm ]);

  const filteredVisibleColumns = useMemo(() => {
    return visibleColumns.filter(column => column.toLowerCase().includes(searchTerm));
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

  const renderColumnName = useCallback((columnName:string) => {
    return columnName === 'id' ? 'ID' : camelCaseToSentence(columnName);
  }, []);

  const generateModalContent = useCallback((state: ModalState): React.ReactNode => {
    // We always render the form regardless of mode to provide a reference to it.
    return (
      <div className={css.base}>
        <Input placeholder="Search columns..." onChange={handleSearch} />
        <div className={css.columns}>
          <div className={css.column}>
            <h2>Hidden</h2>
            <ul>
              {filteredHiddenColumns.map(column => (
                <li key={column} onClick={() => makeVisible(column)}>
                  {renderColumnName(column)}
                </li>
              ))}
            </ul>
            <Button type="link" onClick={() => makeVisible(filteredHiddenColumns)}>
              Add All
            </Button>
          </div>
          <div className={css.column}>
            <h2>Visible</h2>
            <ul>
              {filteredVisibleColumns.map(column => (
                <li key={column} onClick={() => makeHidden(column)}>
                  {renderColumnName(column)}
                </li>
              ))}
            </ul>
            <Button type="link" onClick={() => makeHidden(filteredVisibleColumns)}>
              Remove All
            </Button>
          </div>
        </div>
      </div>
    );
  }, [ handleSearch,
    filteredHiddenColumns,
    filteredVisibleColumns,
    renderColumnName,
    makeVisible,
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
