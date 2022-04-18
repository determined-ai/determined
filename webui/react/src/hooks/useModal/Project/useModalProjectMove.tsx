import { Select } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import { SelectValue } from 'antd/lib/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import SelectFilter from 'components/SelectFilter';
import useModal, { ModalHooks } from 'hooks/useModal/useModal';
import { getWorkspaces, moveProject } from 'services/api';
import { Workspace } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

import css from './useModalProjectMove.module.scss';

const { Option } = Select;

interface Props {
  onClose?: () => void;
  projectId: number;
}

const useModalProjectMove = ({ onClose, projectId }: Props): ModalHooks => {
  const [ destinationWorkspaceId, setDestinationWorkspaceId ] = useState<number>();
  const [ workspaces, setWorkspaces ] = useState<Workspace[]>([]);

  const handleClose = useCallback(() => {
    onClose?.();
  }, [ onClose ]);

  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal({ onClose: handleClose });

  const fetchWorkspaces = useCallback(async () => {
    try {
      const response = await getWorkspaces({ limit: 0 });
      setWorkspaces(response.workspaces);
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to fetch workspaces.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, []);

  useEffect(() => {
    fetchWorkspaces();
  }, [ fetchWorkspaces ]);

  const handleWorkspaceSelect = useCallback((value: SelectValue) => {
    setDestinationWorkspaceId(value as number);
  }, []);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <label className={css.label} htmlFor="workspace">Workspace</label>
        <SelectFilter
          id="workspace"
          placeholder="Select a destination workspace."
          style={{ width: '100%' }}
          onChange={handleWorkspaceSelect}>
          {workspaces.map(workspace => {
            return (
              <Option key={workspace.id} value={workspace.id}>
                {workspace.name}
              </Option>
            );
          })}
        </SelectFilter>
      </div>
    );
  }, [ handleWorkspaceSelect, workspaces ]);

  const handleOk = useCallback(async () => {
    if (!destinationWorkspaceId) return;
    try {
      await moveProject({ destinationWorkspaceId, projectId });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to move project.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ destinationWorkspaceId, projectId ]);

  const getModalProps = useCallback((destinationWorkspaceId?: number): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { disabled: !destinationWorkspaceId },
      okText: 'Move Project',
      onOk: handleOk,
      title: 'Move Project',
    };
  }, [ handleOk, modalContent ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    setDestinationWorkspaceId(undefined);
    openOrUpdate({ ...getModalProps(undefined), ...initialModalProps });
  }, [ getModalProps, openOrUpdate ]);

  /*
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps(destinationWorkspaceId));
  }, [ destinationWorkspaceId, getModalProps, modalRef, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalProjectMove;
