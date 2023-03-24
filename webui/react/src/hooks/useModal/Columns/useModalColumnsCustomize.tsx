import { ModalFuncProps } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Transfer from 'components/Transfer';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';
import usePrevious from 'shared/hooks/usePrevious';
import { isEqual } from 'shared/utils/data';

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
  const prevInitVisibleColumns = usePrevious(initialVisibleColumns, defaultVisibleColumns);
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal();
  const [visibleColumns, setVisibleColumns] = useState<string[]>(
    initialVisibleColumns ?? defaultVisibleColumns,
  );
  const [modalVisible, setModalVisible] = useState(false);

  useEffect(() => {
    // If you travel between pages that both use this hook `visibleColumns` doesn't reinitialize.
    // That means that when you open this modal on the second page it will have the value of `visibleColumns` from the first page.
    // This useEffect makes sure that if the value of `initialVisibleColumns` prop changes that
    // `visibleColumns` will be set to the new value.
    if (!isEqual(initialVisibleColumns, prevInitVisibleColumns))
      setVisibleColumns(initialVisibleColumns ?? defaultVisibleColumns);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialVisibleColumns]);

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
        setModalVisible(false);
      },
      title: 'Customize Columns',
    };
  }, [modalContent, onSave, visibleColumns]);

  const modalOpen = useCallback(
    ({ initialModalProps }: ShowModalProps) => {
      setModalVisible(true);
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
    if (modal && modalVisible) openOrUpdate(modalProps);
  }, [modalProps, modalRef, modalVisible, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalColumnsCustomize;
