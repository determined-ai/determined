import React, { useCallback, useMemo } from 'react';

import css from 'components/ActionDropdown/ActionDropdown.module.scss';
import Button from 'components/kit/Button';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import usePermissions from 'hooks/usePermissions';
import { archiveProject, unarchiveProject } from 'services/api';
import { Project } from 'types';
import handleError from 'utils/error';

import ProjectDeleteModalComponent from './ProjectDeleteModal';
import ProjectEditModalComponent from './ProjectEditModal';
import ProjectMoveModalComponent from './ProjectMoveModal';

interface Props {
  children?: React.ReactNode;
  className?: string;
  direction?: 'vertical' | 'horizontal';
  isContextMenu?: boolean;
  onDelete?: () => void;
  onEdit?: (name: string, archived: boolean) => void;
  onMove?: () => void;
  project: Project;
  showChildrenIfEmpty?: boolean;
  workspaceArchived?: boolean;
}

interface ProjectMenuPropsIn {
  onDelete?: () => void;
  onEdit?: (name: string, archived: boolean) => void;
  onMove?: () => void;
  project?: Project;
  workspaceArchived?: boolean;
}

interface ProjectMenuPropsOut {
  contextHolders: React.ReactElement;
  menu: MenuItem[];
  onClick: (key: string) => void;
}

export const useProjectActionMenu: (props: ProjectMenuPropsIn) => ProjectMenuPropsOut = ({
  onDelete,
  onEdit,
  onMove,
  project,
  workspaceArchived = false,
}: ProjectMenuPropsIn) => {
  const ProjectMoveModal = useModal(ProjectMoveModalComponent);
  const ProjectDeleteModal = useModal(ProjectDeleteModalComponent);
  const ProjectEditModal = useModal(ProjectEditModalComponent);

  const contextHolders = useMemo(() => {
    return (
      <>
        {project && (
          <>
            <ProjectMoveModal.Component project={project} onMove={onMove} />
            <ProjectDeleteModal.Component project={project} onDelete={onDelete} />
            <ProjectEditModal.Component project={project} onEdit={onEdit} />
          </>
        )}
      </>
    );
  }, [ProjectMoveModal, ProjectEditModal, ProjectDeleteModal, onDelete, onEdit, onMove, project]);

  const { canDeleteProjects, canModifyProjects, canMoveProjects } = usePermissions();

  const handleArchiveClick = useCallback(async () => {
    if (!project) return;

    if (project.archived) {
      try {
        await unarchiveProject({ id: project.id });
        onEdit?.(project.name, false);
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to unarchive project.' });
      }
    } else {
      try {
        await archiveProject({ id: project.id });
        onEdit?.(project.name, true);
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to archive project.' });
      }
    }
  }, [onEdit, project]);

  const MenuKey = {
    Delete: 'delete',
    Edit: 'edit',
    Move: 'move',
    SwitchArchived: 'switchArchive',
  } as const;

  const items: MenuItem[] = [];
  if (project && !project.immutable) {
    if (
      canModifyProjects({ project, workspace: { id: project.workspaceId } }) &&
      !project.archived
    ) {
      items.push({ key: MenuKey.Edit, label: 'Edit...' });
    }
    if (canMoveProjects({ project }) && !project.archived) {
      items.push({ key: MenuKey.Move, label: 'Move...' });
    }
    if (
      canModifyProjects({ project, workspace: { id: project.workspaceId } }) &&
      !workspaceArchived
    ) {
      const label = project.archived ? 'Unarchive' : 'Archive';
      items.push({ key: MenuKey.SwitchArchived, label: label });
    }
    if (
      canDeleteProjects({ project, workspace: { id: project.workspaceId } }) &&
      !project.archived &&
      project.numExperiments === 0
    ) {
      items.push({ danger: true, key: MenuKey.Delete, label: 'Delete...' });
    }
  }

  const handleDropdown = (key: string) => {
    switch (key) {
      case MenuKey.Delete:
        ProjectDeleteModal.open();
        break;
      case MenuKey.Edit:
        ProjectEditModal.open();
        break;
      case MenuKey.Move:
        ProjectMoveModal.open();
        break;
      case MenuKey.SwitchArchived:
        handleArchiveClick();
        break;
    }
  };

  return { contextHolders, menu: items, onClick: handleDropdown };
};

const ProjectActionDropdown: React.FC<Props> = ({
  project,
  children,
  isContextMenu,
  showChildrenIfEmpty = true,
  className,
  direction = 'vertical',
  onEdit,
  onDelete,
  onMove,
  workspaceArchived = false,
}: Props) => {
  const { contextHolders, menu, onClick } = useProjectActionMenu({
    onDelete,
    onEdit,
    onMove,
    project,
    workspaceArchived,
  });

  if (menu?.length === 0 && !showChildrenIfEmpty) {
    return null;
  }

  return children ? (
    <>
      <Dropdown
        disabled={menu?.length === 0}
        isContextMenu={isContextMenu}
        menu={menu}
        onClick={onClick}>
        {children}
      </Dropdown>
      {contextHolders}
    </>
  ) : (
    <div className={[css.base, className].join(' ')} title="Open actions menu">
      <Dropdown disabled={menu?.length === 0} menu={menu} placement="bottomRight" onClick={onClick}>
        <Button icon={<Icon name={`overflow-${direction}`} title="Action menu" />} />
      </Dropdown>
      {contextHolders}
    </div>
  );
};

export default ProjectActionDropdown;
