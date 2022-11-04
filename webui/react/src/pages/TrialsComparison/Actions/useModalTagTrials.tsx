import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import TagList from 'components/Tags';
import {
  getDescriptionText,
  isTrialsCollection,
  TrialsSelectionOrCollection,
} from 'pages/TrialsComparison/Collections/collections';
import { updateTrialTags } from 'services/api';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';
import { ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

import { encodeFilters, encodeIdList } from '../api';

import css from './useModalTagTrials.module.scss';

interface Props {
  onClose?: () => void;
  onConfirm?: () => void;
}

export interface ShowModalProps {
  initialModalProps?: ModalFuncProps;
  trials: TrialsSelectionOrCollection;
}
interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (props: ShowModalProps) => void;
}

const useModalTrialTag = ({ onClose, onConfirm }: Props): ModalHooks => {
  const [trials, setTrials] = useState<TrialsSelectionOrCollection>({ trialIds: [] });
  const [tags, setTags] = useState<string[]>([]);
  const handleClose = useCallback(() => onClose?.(), [onClose]);

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
  }, [tags]);

  const handleOk = useCallback(
    async (trials) => {
      const patch = {
        addTag: tags.map((tag) => {
          return { key: tag };
        }),
      };
      const target = isTrialsCollection(trials)
        ? { filters: encodeFilters(trials.filters) }
        : { trial: { ids: encodeIdList(trials.trialIds) } };
      try {
        await updateTrialTags({ patch, ...target });
      } catch (e) {
        handleError(e, {
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to add tags.',
          silent: false,
          type: ErrorType.Api,
        });
      }
      onConfirm?.();
    },
    [tags, onConfirm],
  );

  const getModalProps = useCallback(
    (trials: TrialsSelectionOrCollection, tags: string[]): ModalFuncProps => {
      return {
        closable: true,
        content: modalContent,
        icon: null,
        okButtonProps: { disabled: tags.length === 0 },
        okText: 'Add Tags',
        onOk: () => handleOk(trials),
        title: `Add tags to ${getDescriptionText(trials)}`,
      };
    },
    [handleOk, modalContent],
  );

  const modalOpen = useCallback(
    ({ initialModalProps, trials }: ShowModalProps) => {
      openOrUpdate({
        ...initialModalProps,
        ...getModalProps(trials, []),
      });
      setTrials(trials);
      setTags([]);
    },
    [getModalProps, openOrUpdate],
  );

  useEffect(() => {
    if (modalRef.current) {
      openOrUpdate(getModalProps(trials, tags));
    }
  }, [getModalProps, modalRef, openOrUpdate, trials, tags]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalTrialTag;
