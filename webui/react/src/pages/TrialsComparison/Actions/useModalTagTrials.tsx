import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useMemo, useState } from 'react';

import TagList from 'components/TagList';
import {
  getDescriptionText,
  isTrialsCollection,
  TrialsSelectionOrCollection,
} from 'pages/TrialsComparison/Collections/collections';
import { patchTrials } from 'services/api';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';

import css from './useModalTagTrials.module.scss';

interface Props {
  onClose?: () => void;
}

export interface ShowModalProps {
  initialModalProps?: ModalFuncProps;
  trials: TrialsSelectionOrCollection;
}
interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (props: ShowModalProps) => void;
}

const useModalTrialTag = ({ onClose }: Props): ModalHooks => {
  const [ tags, setTags ] = useState<string[]> ([]);
  const handleClose = useCallback(() => onClose?.(), [ onClose ]);

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose: handleClose });

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        Tags
        <TagList
          ghost={false}
          tags={tags}
          onChange={(newTags) => {
            setTags(newTags);
          }}
        />
      </div>
    );
  }, [ tags ]);

  const handleOk = useCallback(async (trials) => {
    const addTags = { addTag: tags.map((tag) => { return { key: tag, value: '1' }; }) };
    const patch = isTrialsCollection(trials) ? { ...addTags, filters: trials.filters } : { ...addTags, trials };
    try {
      await patchTrials({ patch });
    } catch (error) {
      // duly noted
    }
  }, [ tags ]);

  const getModalProps = useCallback((trials: TrialsSelectionOrCollection) : ModalFuncProps => {

    return {
      closable: true,
      content: modalContent,
      icon: null,
      okText: 'Add Tags',
      onOk: handleOk,
      title: `Add tags to ${getDescriptionText(trials)}`,
    };
  }, [ handleOk, modalContent ]);

  const modalOpen = useCallback(
    ({
      initialModalProps,
      trials,
    }: ShowModalProps) => {
      openOrUpdate({
        ...initialModalProps,
        ...getModalProps(trials),
      });
    },
    [
      getModalProps,
      openOrUpdate,
    ],
  );

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalTrialTag;
