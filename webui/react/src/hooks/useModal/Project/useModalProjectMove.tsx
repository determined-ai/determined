import { Form, notification, Select, Typography } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Link from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { getWorkspaces, moveProject } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { isEqual } from 'shared/utils/data';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { Project, Workspace } from 'types';
import handleError from 'utils/error';

import css from './useModalProjectMove.module.scss';

const { Option } = Select;

const FORM_ID = 'move-project-form';

interface FormInputs {
  destinationWorkspaceId: number;
}

interface Props {
  onClose?: () => void;
  project: Project;
}

const useModalProjectMove = ({ onClose, project }: Props): ModalHooks => {
  const [form] = Form.useForm<FormInputs>();
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const { canMoveProjectsTo } = usePermissions();

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });

  const fetchWorkspaces = useCallback(async () => {
    try {
      const response = await getWorkspaces({ limit: 0 });
      setWorkspaces((prev) => {
        const withoutDefault = response.workspaces.filter(
          (w) => !w.immutable && canMoveProjectsTo({ destination: { id: w.id } }),
        );
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
  }, [canMoveProjectsTo]);

  useEffect(() => {
    if (modalRef.current) fetchWorkspaces();
  }, [fetchWorkspaces, modalRef]);

  const modalContent = useMemo(() => {
    return (
      <Form autoComplete="off" className={css.base} form={form} id={FORM_ID} layout="vertical">
        <Form.Item
          label="Workspace"
          name="destinationWorkspaceId"
          rules={[{ message: 'Workspace is required', required: true }]}>
          <Select
            placeholder="Select a destination workspace."
            showSearch={false}
            style={{ width: '100%' }}>
            {workspaces.map((workspace) => {
              const disabled = workspace.archived || workspace.id === project.workspaceId;
              return (
                <Option disabled={disabled} key={workspace.id} value={workspace.id}>
                  <div className={disabled ? css.workspaceOptionDisabled : ''}>
                    <Typography.Text ellipsis={true}>{workspace.name}</Typography.Text>
                    {workspace.archived && <Icon name="archive" />}
                    {workspace.id === project.workspaceId && <Icon name="checkmark" />}
                  </div>
                </Option>
              );
            })}
          </Select>
        </Form.Item>
      </Form>
    );
  }, [form, workspaces, project.workspaceId]);

  const handleOk = useCallback(async () => {
    const values = await form.validateFields();
    const destinationWorkspaceId = values.destinationWorkspaceId;

    try {
      await moveProject({ destinationWorkspaceId, projectId: project.id });
      const destinationWorkspaceName: string =
        workspaces.find((w) => w.id === destinationWorkspaceId)?.name ?? '';
      notification.open({
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
      });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to move project.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [form, project.id, project.name, workspaces]);

  const getModalProps = useMemo((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { form: FORM_ID, htmlType: 'submit' },
      okText: 'Move Project',
      onOk: handleOk,
      title: 'Move Project',
    };
  }, [handleOk, modalContent]);

  const modalOpen = useCallback(
    (initialModalProps: ModalFuncProps = {}) => {
      fetchWorkspaces();
      openOrUpdate({ ...getModalProps, ...initialModalProps });
    },
    [fetchWorkspaces, getModalProps, openOrUpdate],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps);
  }, [getModalProps, modalRef, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalProjectMove;
