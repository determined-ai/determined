import { Button, Input, Modal, ModalFuncProps, Space, Transfer } from 'antd';
import { ModalFunc } from 'antd/es/modal/confirm';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { camelCaseToSentence, sentenceToCamelCase } from 'utils/string';

import css from './useCustomizeColumnsModal.module.scss';

interface ModalState {
  columns: string[];
  defaultVisibleColumns: string[];
  visible: boolean;
  visibleColumns: string[];
}

export interface ShowModalProps {
  columns: string[];
  defaultVisibleColumns: string[];
  visibleColumns: string[];
}

interface ModalHooks {
  showModal: (props: ShowModalProps) => void;
}

const useCustomizeColumnsModal = (): ModalHooks => {
  const modalRef = useRef<ReturnType<ModalFunc>>();
  const [ modalState, setModalState ] = useState<ModalState>({
    columns: [],
    defaultVisibleColumns: [],
    visible: false,
    visibleColumns: [],
  });

  const showModal = useCallback((
    { columns, visibleColumns, defaultVisibleColumns }: ShowModalProps,
  ) => {
    setModalState({ columns, defaultVisibleColumns, visible: true, visibleColumns });
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

  const handleSave = useCallback(() => {
    if (!modalRef.current) return;
    closeModal();
  }, [ closeModal ]);

  const hiddenColumns = useMemo(() => {
    return modalState.columns.filter(column =>
      !modalState.visibleColumns.includes(sentenceToCamelCase(column)));
  }, [ modalState.columns, modalState.visibleColumns ]);

  const generateModalContent = useCallback((state: ModalState): React.ReactNode => {
    const { visibleColumns } = state;
    // We always render the form regardless of mode to provide a reference to it.
    return (
      <div className={css.base}>
        <Input placeholder="Search columns..." />
        <div className={css.columns}>
          <div className={css.column}>
            <h2>Hidden</h2>
            <ul>{hiddenColumns.map(column => <li key={column}>{column}</li>)}</ul>
            <Button type="link">Add All</Button>
          </div>
          <div className={css.column}>
            <h2>Visible</h2>
            <ul>{visibleColumns.map(column =>
              <li key={column}>{column === 'id' ? 'ID' : camelCaseToSentence(column)}</li>)}
            </ul>
            <Button type="link">Remove All</Button>
          </div>
        </div>
      </div>
    );
  }, [ hiddenColumns ]);

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
      onOk: () => handleSave(),
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
