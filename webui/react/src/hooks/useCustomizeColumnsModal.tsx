import { Input, Modal, ModalFuncProps, Transfer } from 'antd';
import { ModalFunc } from 'antd/es/modal/confirm';
import { TransferItem } from 'antd/lib/transfer';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import css from './useCreateModelModal.module.scss';

interface ModalState {
  columns: TransferItem[];
  defaultVisibleColumns: string[];
  visible: boolean;
  visibleColumns: string[];
}

export interface ShowModalProps {
  columns: TransferItem[];
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

  const generateModalContent = useCallback((state: ModalState): React.ReactNode => {
    const { columns, visibleColumns } = state;
    // We always render the form regardless of mode to provide a reference to it.
    return (
      <div className={css.base}>
        <Transfer
          dataSource={columns}
          render={item => item.title ?? ''}
          targetKeys={visibleColumns}
        />
      </div>
    );
  }, [ ]);

  const generateModalProps = useCallback((state: ModalState): Partial<ModalFuncProps> => {
    const modalProps = {
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
