import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { setProjectNotes } from 'services/api';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { Project } from 'types';
import handleError from 'utils/error';

import css from './useModalProjectNoteDelete.module.scss';

interface Props {
  onClose?: () => void;
  project?: Project;
}

export interface ShowModalProps {
  initialModalProps?: ModalFuncProps;
  pageNumber: number;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (props: ShowModalProps) => void;
}

const useModalProjectNoteDelete = ({ onClose, project }: Props = {}): ModalHooks => {
  const [pageNumber, setPageNumber] = useState(0);

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <p>
          Are you sure you want to delete&nbsp;
          <strong>&quot;{project?.notes?.[pageNumber]?.name ?? 'Untitled'}&quot;</strong>?
        </p>
        <p>This cannot be undone.</p>
      </div>
    );
  }, [pageNumber, project?.notes]);

  const handleOk = useCallback(async () => {
    if (!project?.id) return;
    try {
      await setProjectNotes({
        notes: project.notes.filter((note, idx) => idx !== pageNumber),
        projectId: project.id,
      });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete notes page.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [pageNumber, project?.id, project?.notes]);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { danger: true },
      okText: 'Delete Page',
      onOk: handleOk,
      title: 'Delete Page',
    };
  }, [handleOk, modalContent]);

  const modalOpen = useCallback(
    ({ pageNumber, initialModalProps }: ShowModalProps) => {
      setPageNumber(pageNumber);
      openOrUpdate({ ...getModalProps(), ...initialModalProps });
    },
    [getModalProps, openOrUpdate],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps());
  }, [getModalProps, modalRef, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalProjectNoteDelete;
