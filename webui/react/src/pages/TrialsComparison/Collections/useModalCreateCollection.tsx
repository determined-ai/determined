import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import useModal, { ModalHooks as Hooks } from 'hooks/useModal/useModal';
import { getDescriptionText } from 'pages/TrialsComparison/Collections/collections';
import { createTrialsCollection, updateTrialTags } from 'services/api';
import { DetError, ErrorType } from 'utils/error';
import handleError from 'utils/error';

import { encodeFilters, encodeTrialSorter } from '../api';

import { isTrialsSelection, TrialsCollection, TrialsSelectionOrCollection } from './collections';
import css from './useModalCreateCollection.module.scss';

interface Props {
  onClose?: () => void;
  onConfirm?: (newCollectionName: string) => Promise<void>;
  projectId: string;
}

export interface CollectionModalProps {
  initialModalProps?: ModalFuncProps;
  trials?: TrialsSelectionOrCollection;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (props: CollectionModalProps) => void;
}

interface FormInputs {
  collectionName: string;
}

const useModalTrialCollection = ({ projectId, onConfirm }: Props): ModalHooks => {
  const [form] = Form.useForm<FormInputs>();
  const [trials, setTrials] = useState<TrialsSelectionOrCollection>();

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal();

  const handleOk = useCallback(
    async (target: TrialsSelectionOrCollection) => {
      const values = await form.validateFields();
      const name = values.collectionName;

      let newCollection: TrialsCollection | undefined;
      try {
        if (isTrialsSelection(target)) {
          await updateTrialTags({
            patch: { addTag: [{ key: name }] },
            trial: { ids: target.trialIds },
          });
          newCollection = await createTrialsCollection({
            filters: encodeFilters({ tags: [name] }),
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
      } catch (e) {
        if (e instanceof DetError) {
          handleError(e, {
            level: e.level,
            publicMessage: e.publicMessage,
            publicSubject: 'Unable to create model.',
            silent: false,
            type: e.type,
          });
        } else {
          handleError(e, {
            publicMessage: 'Please try again later.',
            publicSubject: 'Unable to create model.',
            silent: false,
            type: ErrorType.Api,
          });
        }
      }
      form.resetFields();
      if (newCollection?.name) onConfirm?.(newCollection.name);
      modalRef.current?.destroy();
      modalRef.current = undefined;
    },
    [form, onConfirm, modalRef, projectId],
  );

  const modalContent = useMemo(() => {
    return (
      <Form autoComplete="off" className={css.base} form={form} layout="vertical">
        <Form.Item
          name="collectionName"
          rules={[{ message: 'Collection name is required ', required: true }]}>
          <Input
            allowClear
            bordered={true}
            placeholder="enter collection name"
            onPressEnter={() => trials && handleOk(trials)}
          />
        </Form.Item>
      </Form>
    );
  }, [form, trials, handleOk]);

  const getModalProps = useCallback(
    (trials?: TrialsSelectionOrCollection): ModalFuncProps => {
      if (!trials) throw Error('trials should not be undefined');
      const actionText = isTrialsSelection(trials) ? 'Tag and Collect' : 'Create Collection for';
      const props = {
        closable: true,
        content: modalContent,
        icon: null,
        okText: 'Create Collection',
        onOk: () => handleOk(trials),
        title: trials && `${actionText} ${getDescriptionText(trials)}`,
      };
      return props;
    },
    [handleOk, modalContent],
  );

  const modalOpen = useCallback(
    ({ initialModalProps, trials }: CollectionModalProps) => {
      setTrials(trials);
      const newProps = {
        ...initialModalProps,
        ...getModalProps(trials),
      };
      openOrUpdate(newProps);
    },
    [getModalProps, openOrUpdate],
  );

  useEffect(() => {
    if (modalRef.current) {
      openOrUpdate(getModalProps(trials));
    }
  }, [getModalProps, modalRef, openOrUpdate, trials]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalTrialCollection;
