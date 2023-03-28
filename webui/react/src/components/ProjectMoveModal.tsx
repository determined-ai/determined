import { Select, Typography } from 'antd';
import { SelectValue } from 'antd/lib/select';
import React, { useCallback, useEffect, useState } from 'react';

import { Modal } from 'components/kit/Modal';
import Link from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { getWorkspaces, moveProject } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import { isEqual } from 'shared/utils/data';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { Project, Workspace } from 'types';
import { notification } from 'utils/dialogApi';
import handleError from 'utils/error';

import css from './ProjectMoveModal.module.scss';

const { Option } = Select;

interface Props {
  onClose?: () => void;
  project: Project;
}

const ProjectMoveModalComponent: React.FC<Props> = ({ onClose, project }: Props) => {
  const [destinationWorkspaceId, setDestinationWorkspaceId] = useState<number>();
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const { canMoveProjectsTo } = usePermissions();

  const handleSubmit = useCallback(async () => {
    if (!destinationWorkspaceId) return;
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
  }, [destinationWorkspaceId, project.id, project.name, workspaces]);

  const fetchWorkspaces = useCallback(async () => {
    try {
      const response = await getWorkspaces({ archived: false, limit: 0 });
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
    fetchWorkspaces();
  }, [fetchWorkspaces]);

  const handleWorkspaceSelect = useCallback(
    (selectedWorkspaceId: SelectValue) => {
      if (typeof selectedWorkspaceId !== 'number') return;
      const workspace = workspaces.find((w) => w.id === selectedWorkspaceId);
      if (!workspace) return;
      const disabled = workspace.archived || workspace.id === project.workspaceId;
      if (disabled) return;
      setDestinationWorkspaceId((prev) => (disabled ? prev : (selectedWorkspaceId as number)));
    },
    [workspaces, project.workspaceId],
  );

  return (
    <Modal
      cancel
      submit={{
        disabled: !destinationWorkspaceId,
        handler: handleSubmit,
        text: 'Move Project',
      }}
      title="Move Project"
      onClose={onClose}>
      <label htmlFor="workspace">Workspace</label>
      <Select
        id="workspace"
        placeholder="Select a destination workspace."
        style={{ width: '100%' }}
        value={destinationWorkspaceId}
        onSelect={handleWorkspaceSelect}>
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
    </Modal>
  );
};

export default ProjectMoveModalComponent;
