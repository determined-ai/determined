import { Select, Typography } from 'antd';
import { SelectValue } from 'antd/lib/select';
import React, { useCallback, useState } from 'react';

import Icon from 'components/kit/Icon';
import { Modal } from 'components/kit/Modal';
import { makeToast } from 'components/kit/Toast';
import { Loadable } from 'components/kit/utils/loadable';
import Link from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { moveProject } from 'services/api';
import workspaceStore from 'stores/workspaces';
import { Project, Workspace } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { useObservable } from 'utils/observable';

import css from './ProjectMoveModal.module.scss';

const { Option } = Select;

interface Props {
  onMove?: () => void;
  project: Project;
}

const ProjectMoveModalComponent: React.FC<Props> = ({ onMove, project }: Props) => {
  const [destinationWorkspaceId, setDestinationWorkspaceId] = useState<number>();
  const { canMoveProjectsTo } = usePermissions();
  const workspaces = Loadable.match(useObservable(workspaceStore.unarchived), {
    _: () => [],
    Loaded: (workspaces: Workspace[]) =>
      workspaces.filter((w) => !w.immutable && canMoveProjectsTo({ destination: { id: w.id } })),
  });

  const handleSubmit = useCallback(async () => {
    if (!destinationWorkspaceId) return;
    try {
      await moveProject({ destinationWorkspaceId, projectId: project.id });
      const destinationWorkspaceName: string =
        workspaces.find((w) => w.id === destinationWorkspaceId)?.name ?? '';
      makeToast({
        description: `${project.name} moved to workspace ${destinationWorkspaceName}`,
        link: <Link path={paths.workspaceDetails(destinationWorkspaceId)}>View Workspace</Link>,
        title: 'Move Success',
      });
      onMove?.();
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to move project.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [destinationWorkspaceId, onMove, project.id, project.name, workspaces]);

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
        handleError,
        handler: handleSubmit,
        text: 'Move Project',
      }}
      title="Move Project">
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
                {workspace.archived && <Icon name="archive" title="Archived" />}
                {workspace.id === project.workspaceId && (
                  <Icon name="checkmark" title="Project's current workspace" />
                )}
              </div>
            </Option>
          );
        })}
      </Select>
    </Modal>
  );
};

export default ProjectMoveModalComponent;
