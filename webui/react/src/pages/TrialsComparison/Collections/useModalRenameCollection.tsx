import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';

import Input from 'components/kit/Input';
import useModal, { ModalHooks as Hooks } from 'hooks/useModal/useModal';
import { patchTrialsCollection } from 'services/api';
import { ErrorLevel, ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { validateLength } from 'utils/string';

import css from './useModalRenameCollection.module.scss';

export interface ShowModalProps {
  id: string;

  initialModalProps?: ModalFuncProps;
  name: string;
}
interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (props: ShowModalProps) => void;
}

interface Props {
  onComplete: (name: string) => void;
}

const useModalRenameCollection = ({ onComplete }: Props): ModalHooks => {
  const [name, setName] = useState('');
  const [collectionId, setCollectionId] = useState<string>();

  const { modalOpen: openOrUpdate, modalRef, ...modalHooks } = useModal();

  const handleNameInput = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setName(e.target.value);
  }, []);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <div>
          <label className={css.label} htmlFor="name">
            Name
          </label>
          <Input id="name" value={name} onChange={handleNameInput} />
        </div>
      </div>
    );
  }, [handleNameInput, name]);

  const handleOk = useCallback(async () => {
    try {
      if (collectionId) {
        await patchTrialsCollection({ id: parseInt(collectionId), name });
        onComplete(name);
      }
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to rename collection.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [name, collectionId, onComplete]);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { disabled: !validateLength(name) },
      okText: 'Save',
      onOk: handleOk,
      title: 'Rename Collection',
    };
  }, [handleOk, modalContent, name]);

  const modalOpen = useCallback(
    ({ initialModalProps, id, name }: ShowModalProps) => {
      setCollectionId(id);
      setName(name);
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
  }, [getModalProps, modalRef, name, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHooks };
};

export default useModalRenameCollection;
