import { Input } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import {
  getDescriptionText,
} from 'pages/TrialsComparison/Collections/collections';
import { createTrialsCollection, patchTrials } from 'services/api';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';

import { encodeFilters, encodeTrialSorter } from '../api';

import { isTrialsSelection, TrialsCollection, TrialsSelectionOrCollection } from './collections';
import css from './useModalCreateCollection.module.scss';

interface Props {
  onClose?: () => void;
  onConfirm?: (newCollection: TrialsCollection) => void;
  projectId: string;

}

export interface CollectionModalProps {
  initialModalProps?: ModalFuncProps;
  trials?: TrialsSelectionOrCollection
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (props: CollectionModalProps) => void;
}

const useModalTrialCollection = ({
  projectId,
  onConfirm,
}: Props): ModalHooks => {

  const [ trials, setTrials ] = useState<TrialsSelectionOrCollection>();
  const [ name, setName ] = useState('');
  const handleNameChange = useCallback((e) => setName(e.target.value), []);

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal();

  const handleOk = useCallback(
    async (target: TrialsSelectionOrCollection) => {
      // const name = inputRef.current?.input?.value;
      if (!name) return;
      let newCollection: TrialsCollection | undefined;
      try {
        if (isTrialsSelection(target)) {

          await patchTrials(
            {
              patch: { addTag: [ { key: name } ] },
              trial: { ids: target.trialIds },
            },
          );
          newCollection = await createTrialsCollection({
            filters: encodeFilters({ tags: [ name ] }),
            name,
            projectId: parseInt(projectId),
            sorter: encodeTrialSorter(target.sorter),
          });

        } else {
          newCollection = await createTrialsCollection({
            filters: encodeFilters(target.filters),
            name,
            projectId: parseInt(projectId),
            sorter: encodeTrialSorter(target.sorter),
          });
        }

      } catch (error) {
        // duly noted
      }
      setName('');
      if (newCollection) onConfirm?.(newCollection);
      modalRef.current?.destroy();
      modalRef.current = undefined;
    },
    [ projectId, onConfirm, name, modalRef ],
  );

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <Input
          allowClear
          bordered={true}
          placeholder="enter collection name"
          value={name}
          onChange={handleNameChange}
          onPressEnter={() => trials && handleOk(trials)}
        />
      </div>
    );
  }, [ name, handleNameChange, handleOk, trials ]);

  const getModalProps = useCallback(
    (trials, name: string): ModalFuncProps => {
      const actionText = isTrialsSelection(trials) ? 'Tag and Collect' : 'Create Collection for';
      const props = {
        closable: true,
        content: modalContent,
        icon: null,
        okButtonProps: { disabled: !name },
        okText: 'Create Collection',
        onOk: () => handleOk(trials),
        title: trials && `${actionText} ${getDescriptionText(trials)}`,
      };
      return props;
    },
    [ handleOk, modalContent ],
  );

  const modalOpen = useCallback(
    ({ initialModalProps, trials }: CollectionModalProps) => {

      setTrials(trials);
      const newProps = {
        ...initialModalProps,
        ...getModalProps(trials, name),
      };
      openOrUpdate(newProps);
    },
    [ getModalProps, openOrUpdate, name ],
  );

  useEffect(() => {
    if (modalRef.current){
      openOrUpdate(getModalProps(trials, name));
    }
  }, [ getModalProps, modalRef, openOrUpdate, trials, name ]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalTrialCollection;
