import { ModalFuncProps } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Transfer from 'components/Transfer';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';

import css from './useModalColumnsCustomize.module.scss';

interface Props {
  columns: string[];
  defaultVisibleColumns: string[];
  initialVisibleColumns?: string[];
  onSave?: (columns: string[]) => void;
}

export interface ShowModalProps {
  initialModalProps?: ModalFuncProps;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (props: ShowModalProps) => void;
}

const useModalColumnsCustomize = ({
  columns,
  defaultVisibleColumns,
  initialVisibleColumns,
  onSave,
}: Props): ModalHooks => {
  const columnList = useRef(columns).current; // This is only to prevent rerendering
  const { modalOpen: openOrUpdate, modalClose, modalRef, ...modalHook } = useModal();
  const [visibleColumns, setVisibleColumns] = useState<string[]>(
    initialVisibleColumns ?? defaultVisibleColumns,
  );

  const modalContent = useMemo((): React.ReactNode => {
    return (
      <Transfer
        defaultTargetEntries={defaultVisibleColumns}
        entries={columnList}
        initialTargetEntries={visibleColumns}
        sourceListTitle="Hidden"
        targetListTitle="Visible"
        onChange={setVisibleColumns}
      />
    );
  }, [defaultVisibleColumns, columnList, visibleColumns]);

  const modalProps: Partial<ModalFuncProps> = useMemo(() => {
    return {
      className: css.base,
      closable: true,
      content: modalContent,
      icon: null,
      maskClosable: true,
      okText: 'Save',
      onOk: () => {
        onSave?.(visibleColumns);
        modalClose();
      },
      title: 'Customize Columns',
    };
  }, [modalContent, onSave, visibleColumns, modalClose]);

  const modalOpen = useCallback(
    ({ initialModalProps }: ShowModalProps) => {
      openOrUpdate({ ...modalProps, ...initialModalProps });
    },
    [modalProps, openOrUpdate],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    const modal = modalRef.current;
    if (modal) openOrUpdate(modalProps);
  }, [modalProps, modalRef, openOrUpdate]);

  return { modalClose, modalOpen, modalRef, ...modalHook };
};

export default useModalColumnsCustomize;
