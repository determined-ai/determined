import Form from 'hew/Form';
import Icon from 'hew/Icon';
import { Modal } from 'hew/Modal';
import Select, { Option, SelectValue } from 'hew/Select';
import { useToast } from 'hew/Toast';
import { Label } from 'hew/Typography';
import { Loadable } from 'hew/utils/loadable';
import React, { useCallback, useState } from 'react';

import Link from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { moveProject } from 'services/api';
import workspaceStore from 'stores/workspaces';
import { Project, Workspace } from 'types';
import handleError, { DetError, ErrorLevel, ErrorType } from 'utils/error';
import { useObservable } from 'utils/observable';

import css from './ProjectMoveModal.module.scss';

const FORM_ID = 'move-project-form';

interface Props {
  onMove?: () => void;
  project: Project;
}

const ProjectMoveModalComponent: React.FC<Props> = ({ onMove, project }: Props) => {
  const [destinationWorkspaceId, setDestinationWorkspaceId] = useState<number>();
  const { canMoveProjectsTo } = usePermissions();
  const { openToast } = useToast();
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
      openToast({
        description: `${project.name} moved to workspace ${destinationWorkspaceName}`,
        link: <Link path={paths.workspaceDetails(destinationWorkspaceId)}>View Workspace</Link>,
        title: 'Move Success',
      });
      onMove?.();
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage:
          e instanceof DetError && e.type === ErrorType.Server
            ? e.publicMessage
            : 'Please try again later.',
        publicSubject: 'Unable to move project.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [destinationWorkspaceId, onMove, openToast, project.id, project.name, workspaces]);

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
        form: FORM_ID,
        handleError,
        handler: handleSubmit,
        text: 'Move Project',
      }}
      title="Move Project">
      <label htmlFor="workspace">Workspace</label>
      <Form id={FORM_ID}>
        <Select
          id="workspace"
          placeholder="Select a destination workspace."
          value={destinationWorkspaceId}
          width={'100%'}
          onSelect={handleWorkspaceSelect}>
          {workspaces.map((workspace) => {
            const disabled = workspace.archived || workspace.id === project.workspaceId;
            return (
              <Option
                disabled={disabled}
                key={workspace.id}
                label={workspace.name}
                value={workspace.id}>
                <div className={disabled ? css.workspaceOptionDisabled : ''}>
                  <Label truncate={{ tooltip: true }}>{workspace.name}</Label>
                  {workspace.archived && <Icon name="archive" title="Archived" />}
                  {workspace.id === project.workspaceId && (
                    <Icon name="checkmark" title="Project's current workspace" />
                  )}
                </div>
              </Option>
            );
          })}
        </Select>
      </Form>
    </Modal>
  );
};

export default ProjectMoveModalComponent;
