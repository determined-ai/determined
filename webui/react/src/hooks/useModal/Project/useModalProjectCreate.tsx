import { Input } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { debounce } from 'throttle-debounce';

import { paths } from 'routes/utils';
import { createProject, getWorkspaceProjects } from 'services/api';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { validateLength } from 'shared/utils/string';
import handleError from 'utils/error';

import css from './useModalProjectCreate.module.scss';

interface Props {
  onClose?: () => void;
  workspaceId: number;
}

const useModalProjectCreate = ({ onClose, workspaceId }: Props): ModalHooks => {
  const [ name, setName ] = useState('');
  const [ description, setDescription ] = useState('');
  const [ isNameUnique, setIsNameUnique ] = useState<boolean>(true);

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });

  const findIsNameUnique = useCallback(async (name: string) => {
    const projectList = await getWorkspaceProjects({ id: workspaceId });
    const duplicateNames = projectList.projects
      .map((project) => project.name)
      .filter((projectName) => name === projectName);
    setIsNameUnique(duplicateNames.length === 0 || name === '');
  }, [ workspaceId ]);

  const handleNameInput = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setName(e.target.value);
    debounce(250, () => findIsNameUnique(e.target.value))();
  }, [ findIsNameUnique ]);

  const handleDescriptionInput = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setDescription(e.target.value);
  }, []);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <div>
          <label className={css.label} htmlFor="name">Name</label>
          <Input id="name" maxLength={80} onChange={handleNameInput} />
          {!isNameUnique &&
            <p className={css.uniqueWarning}>A project with this name already exists</p>
          }
        </div>
        <div>
          <label className={css.label} htmlFor="description">Description</label>
          <Input id="description" onChange={handleDescriptionInput} />
        </div>
      </div>
    );
  }, [ handleDescriptionInput, handleNameInput, isNameUnique ]);

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

  const getModalProps = useCallback((name = ''): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { disabled: !validateLength(name) || !isNameUnique },
      okText: 'Create Project',
      onOk: handleOk,
      title: 'New Project',
    };
  }, [ handleOk, isNameUnique, modalContent ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    setName('');
    setDescription('');
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

export default useModalProjectCreate;
