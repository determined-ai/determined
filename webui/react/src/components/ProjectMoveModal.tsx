import { Select, Typography } from 'antd';
import { SelectValue } from 'antd/lib/select';
import React, { useCallback, useState } from 'react';

import Icon from 'components/kit/Icon';
import { Modal } from 'components/kit/Modal';
import Link from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { moveProject } from 'services/api';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import workspaceStore from 'stores/workspaces';
import { Project, Workspace } from 'types';
import { notification } from 'utils/dialogApi';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import css from './ProjectMoveModal.module.scss';

const { Option } = Select;

interface Props {
  onClose?: () => void;
  project: Project;
}

const ProjectMoveModalComponent: React.FC<Props> = ({ onClose, project }: Props) => {
  const [destinationWorkspaceId, setDestinationWorkspaceId] = useState<number>();
  const { canMoveProjectsTo } = usePermissions();
  const workspaces = Loadable.match(useObservable(workspaceStore.unarchived), {
    Loaded: (workspaces: Workspace[]) =>
      workspaces.filter((w) => !w.immutable && canMoveProjectsTo({ destination: { id: w.id } })),
    NotLoaded: () => [],
  });

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
      size="small"
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
