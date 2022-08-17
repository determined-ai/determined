import { Input } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';

import { CharLength } from 'constants/values';
import { paths } from 'routes/utils';
import { createWorkspace } from 'services/api';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { DetError, ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { validateLength } from 'shared/utils/string';
import handleError from 'utils/error';

import css from './useModalWorkspaceCreate.module.scss';

interface Props {
  onClose?: () => void;
}

const useModalWorkspaceCreate = ({ onClose }: Props = {}): ModalHooks => {
  const [ name, setName ] = useState('');

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });

  const handleNameInput = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setName(e.target.value);
  }, []);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <label className={css.label} htmlFor="name">Name</label>
        {/* Input doesnt have value prop due to cusor jump to the end of text */}
        <Input
          defaultValue=""
          id="name"
          maxLength={CharLength.Limit64}
          onChange={handleNameInput}
        />
      </div>
    );
  }, [ handleNameInput ]);

  const handleOk = useCallback(async () => {
    try {
      const response = await createWorkspace({ name });
      routeToReactUrl(paths.workspaceDetails(response.id));
    } catch (e) {
      if (e instanceof DetError) {
        handleError(e, {
          level: e.level,
          publicMessage: e.publicMessage,
          publicSubject: 'Unable to create workspace.',
          silent: false,
          type: e.type,
        });
      } else {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to create workspace.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    }
  }, [ name ]);

  const getModalProps = useCallback((name = ''): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { disabled: !validateLength(name) },
      okText: 'Create Workspace',
      onOk: handleOk,
      title: 'New Workspace',
    };
  }, [ handleOk, modalContent ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    setName('');
    openOrUpdate({ ...getModalProps(''), ...initialModalProps });
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

export default useModalWorkspaceCreate;
