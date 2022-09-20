import { ModalFuncProps } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

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
  const [columnList] = useState(columns); // This is only to prevent rerendering
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal();
  const [visibleColumns, setVisibleColumns] = useState<string[]>(
    initialVisibleColumns ?? defaultVisibleColumns,
  );

  const handleSave = useCallback(() => {
    onSave?.(visibleColumns);
  }, [onSave, visibleColumns]);

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
      onOk: handleSave,
      title: 'Customize Columns',
    };
  }, [modalContent, handleSave]);

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
    if (modalRef.current) openOrUpdate(modalProps);
  }, [modalProps, modalRef, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalColumnsCustomize;
