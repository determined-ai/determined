import { Input } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';

import useModal, { ModalHooks } from 'hooks/useModal/useModal';
import { patchProject } from 'services/api';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { Project } from 'types';
import handleError from 'utils/error';

import css from './useModalProjectEdit.module.scss';

interface Props {
  onClose?: () => void;
  project: Project;
}

const useModalProjectEdit = ({ onClose, project }: Props): ModalHooks => {
  const [ name, setName ] = useState(project.name);
  const [ description, setDescription ] = useState(project.description ?? '');

  const handleClose = useCallback(() => {
    onClose?.();
  }, [ onClose ]);

  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal({ onClose: handleClose });

  const handleNameInput = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setName(e.target.value);
  }, []);

  const handleDescriptionInput = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setDescription(e.target.value);
  }, []);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <div>
          <label className={css.label} htmlFor="name">Name</label>
          <Input id="name" value={name} onChange={handleNameInput} />
        </div>
        <div>
          <label className={css.label} htmlFor="description">Description</label>
          <Input id="description" value={description} onChange={handleDescriptionInput} />
        </div>
      </div>
    );
  }, [ description, handleDescriptionInput, handleNameInput, name ]);

  const handleOk = useCallback(async () => {
    try {
      await patchProject({ description, id: project.id, name });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to edit project.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ description, name, project.id ]);

  const getModalProps = useCallback((name: string): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { disabled: name.length === 0 },
      okText: 'Save Changes',
      onOk: handleOk,
      title: 'Edit Project',
    };
  }, [ handleOk, modalContent ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    openOrUpdate({ ...getModalProps(project.name), ...initialModalProps });
  }, [ getModalProps, openOrUpdate, project.name ]);

  /*
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps(name));
  }, [ getModalProps, modalRef, name, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalProjectEdit;
