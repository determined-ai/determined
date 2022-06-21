import { Input } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';

import useModal, { ModalHooks } from 'hooks/useModal/useModal';
import { paths } from 'routes/utils';
import { createProject } from 'services/api';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import handleError from 'utils/error';

import css from './useModalProjectCreate.module.scss';

interface Props {
  onClose?: () => void;
  workspaceId: number;
}

const useModalProjectCreate = ({ onClose, workspaceId }: Props): ModalHooks => {
  const [ name, setName ] = useState('');
  const [ description, setDescription ] = useState('');

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
      const response = await createProject({ description, name, workspaceId });
      routeToReactUrl(paths.projectDetails(response.id));
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to create project.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ name, workspaceId, description ]);

  const getModalProps = useCallback((name: string): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { disabled: name.length === 0 },
      okText: 'Create Project',
      onOk: handleOk,
      title: 'New Project',
    };
  }, [ handleOk, modalContent ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    setName('');
    setDescription('');
    openOrUpdate({ ...getModalProps(''), ...initialModalProps });
  }, [ getModalProps, openOrUpdate ]);

  /*
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps(name));
  }, [ getModalProps, modalRef, name, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalProjectCreate;
