import { DeleteOutlined } from '@ant-design/icons';
import { Button, Form, Input } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { deleteExperimentGroup, getExperimentGroups, patchExperimentGroup } from 'services/api';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { alphaNumericSorter } from 'shared/utils/sort';
import { ExperimentGroup } from 'types';
import handleError from 'utils/error';

import css from './useModalExperimentGroups.module.scss';

interface Props {
  onClose?: () => void;
  projectId: number;
}

const useModalExperimentGroups = ({ onClose, projectId }: Props): ModalHooks => {
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });
  const [form] = Form.useForm();
  const [experimentGroups, setExperimentGroups] = useState<ExperimentGroup[]>([]);
  const [canceler] = useState(new AbortController());

  const fetchExperimentGroups = useCallback(async () => {
    try {
      const groups = await getExperimentGroups({ projectId }, { signal: canceler.signal });
      groups.sort((a, b) => alphaNumericSorter(a.name, b.name));
      setExperimentGroups(groups);
    } catch (e) {
      handleError(e);
    }
  }, [canceler.signal, projectId]);

  const handleRenameClick = useCallback(
    async (groupId: number) => {
      try {
        const name: string = form.getFieldValue(`name-${groupId}`);
        await patchExperimentGroup({ groupId, name, projectId }, { signal: canceler.signal });
        await fetchExperimentGroups();
      } catch (e) {
        handleError(e);
      }
    },
    [canceler.signal, fetchExperimentGroups, form, projectId],
  );

  const handleDeleteClick = useCallback(
    async (groupId: number) => {
      try {
        await deleteExperimentGroup({ groupId, projectId }, { signal: canceler.signal });
        await fetchExperimentGroups();
      } catch (e) {
        handleError(e);
      }
    },
    [canceler.signal, fetchExperimentGroups, projectId],
  );

  const modalContent = useMemo(() => {
    return (
      <Form className={css.container} form={form} layout="vertical">
        {experimentGroups.map((group) => (
          <div className={css.groupRow} key={group.id}>
            <span>Name:</span>
            <Form.Item initialValue={group.name} label="Name" name={`name-${group.id}`} noStyle>
              <Input />
            </Form.Item>
            <Button onClick={() => handleRenameClick(group.id)}>Rename</Button>
            <Button
              danger
              icon={<DeleteOutlined />}
              type="text"
              onClick={() => handleDeleteClick(group.id)}
            />
          </div>
        ))}
      </Form>
    );
  }, [experimentGroups, form, handleDeleteClick, handleRenameClick]);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      className: css.base,
      closable: true,
      content: modalContent,
      icon: null,
      title: 'Manage Experiment Groups',
    };
  }, [modalContent]);

  const modalOpen = useCallback(
    (initialModalProps: ModalFuncProps = {}) => {
      form.resetFields();
      fetchExperimentGroups();
      openOrUpdate({ ...getModalProps(), ...initialModalProps });
    },
    [fetchExperimentGroups, form, getModalProps, openOrUpdate],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps());
  }, [getModalProps, modalRef, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalExperimentGroups;
