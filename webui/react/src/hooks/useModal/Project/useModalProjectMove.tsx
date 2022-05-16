import { notification, Select, Typography } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import { SelectValue } from 'antd/lib/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Icon from 'components/Icon';
import Link from 'components/Link';
import useModal, { ModalHooks } from 'hooks/useModal/useModal';
import { paths } from 'routes/utils';
import { getWorkspaces, moveProject } from 'services/api';
import { Project, Workspace } from 'types';
import { isEqual } from 'utils/data';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

import css from './useModalProjectMove.module.scss';

const { Option } = Select;

interface Props {
  onClose?: () => void;
  project: Project;
}

const useModalProjectMove = ({ onClose, project }: Props): ModalHooks => {
  const [ destinationWorkspaceId, setDestinationWorkspaceId ] = useState<number>();
  const [ workspaces, setWorkspaces ] = useState<Workspace[]>([]);

  const handleClose = useCallback(() => {
    onClose?.();
  }, [ onClose ]);

  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal({ onClose: handleClose });

  const fetchWorkspaces = useCallback(async () => {
    try {
      const response = await getWorkspaces({ limit: 0 });
      setWorkspaces(prev => {
        const withoutDefault = response.workspaces.filter(w =>
          !w.immutable);
        if (isEqual(prev, withoutDefault)) return prev;
        return withoutDefault;
      });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to fetch workspaces.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ ]);

  useEffect(() => {
    if (modalRef.current) fetchWorkspaces();
  }, [ fetchWorkspaces, modalRef ]);

  const handleWorkspaceSelect = useCallback((selectedWorkspaceId: SelectValue) => {

    if (typeof selectedWorkspaceId !== 'number') return;
    const workspace = workspaces.find(w => w.id === selectedWorkspaceId);
    if (!workspace) return;
    const disabled = workspace.archived || workspace.id === project.workspaceId;
    if (disabled) return;
    setDestinationWorkspaceId(prev => disabled ? prev : selectedWorkspaceId as number);
  }, [ workspaces, project.workspaceId ]);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <label className={css.label} htmlFor="workspace">Workspace</label>
        <Select
          id="workspace"
          placeholder="Select a destination workspace."
          showSearch={false}
          style={{ width: '100%' }}
          value={destinationWorkspaceId}
          onSelect={handleWorkspaceSelect}>
          {workspaces.map(workspace => {
            const disabled = workspace.archived || workspace.id === project.workspaceId;
            return (
              <Option
                disabled={disabled}
                key={workspace.id}
                value={workspace.id}>
                <div className={disabled ? css.workspaceOptionDisabled : ''}>
                  <Typography.Text
                    ellipsis={true}>
                    {workspace.name}
                  </Typography.Text>
                  {workspace.archived && <Icon name="archive" />}
                  {workspace.id === project.workspaceId && <Icon name="checkmark" />}
                </div>
              </Option>
            );
          })}
        </Select>
      </div>
    );
  }, [ handleWorkspaceSelect, workspaces, project.workspaceId, destinationWorkspaceId ]);

  const handleOk = useCallback(async () => {
    if (!destinationWorkspaceId) return;
    try {
      await moveProject({ destinationWorkspaceId, projectId: project.id });
      const destinationWorkspaceName: string =
        workspaces.find((w) => w.id === destinationWorkspaceId)?.name ?? '';
      notification.open(
        {
          btn: null,
          description: (
            <div>
              <p>
                {project.name} moved to workspace {destinationWorkspaceName}
              </p>
              <Link path={paths.workspaceDetails(destinationWorkspaceId)}>View Workspace</Link>
            </div>
          ),
          message: 'Move Success',
        },
      );
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to move project.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ destinationWorkspaceId, project.id, project.name, workspaces ]);

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
    fetchWorkspaces();
    openOrUpdate({ ...getModalProps(undefined), ...initialModalProps });
  }, [ fetchWorkspaces, getModalProps, openOrUpdate ]);

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
