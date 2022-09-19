import { Input } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';

import { paths } from 'routes/utils';
import { deleteProject } from 'services/api';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { Project } from 'types';
import handleError from 'utils/error';

import css from './useModalProjectDelete.module.scss';

interface Props {
  onClose?: () => void;
  project: Project;
}

const useModalProjectDelete = ({ onClose, project }: Props): ModalHooks => {
  const [ name, setName ] = useState('');

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });

  const handleNameInput = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setName(e.target.value);
  }, []);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <p>Are you sure you want to delete <strong>&quot;{project.name}&quot;</strong>?</p>
        <p>All experiments and notes within it will also be deleted. This cannot be undone.</p>
        <label className={css.label} htmlFor="name">Enter project name to confirm deletion</label>
        <Input id="name" value={name} onChange={handleNameInput} />
      </div>
    );
  }, [ handleNameInput, name, project.name ]);

  const handleOk = useCallback(async () => {
    try {
      await deleteProject({ id: project.id });
      routeToReactUrl(paths.workspaceDetails(project.workspaceId));
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete project.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ project.id, project.workspaceId ]);

  const getModalProps = useCallback((name = ''): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { danger: true, disabled: name !== project.name },
      okText: 'Delete Project',
      onOk: handleOk,
      title: 'Delete Project',
    };
  }, [ handleOk, modalContent, project.name ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    setName('');
    openOrUpdate({ ...getModalProps(), ...initialModalProps });
  }, [ getModalProps, openOrUpdate ]);

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps(name));
  }, [ getModalProps, modalRef, name, openOrUpdate ]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalProjectDelete;
