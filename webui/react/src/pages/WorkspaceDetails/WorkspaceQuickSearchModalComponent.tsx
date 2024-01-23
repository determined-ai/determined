import Icon from 'hew/Icon';
import Input from 'hew/Input';
import Message from 'hew/Message';
import { Modal } from 'hew/Modal';
import Spinner from 'hew/Spinner';
import Tree, { TreeDataNode } from 'hew/Tree';
import React, { useCallback, useMemo, useState } from 'react';

import Link from 'components/Link';
import { paths } from 'routes/utils';
import { Project, Workspace } from 'types';
import { routeToReactUrl } from 'utils/routes';

import css from './WorkspaceQuickSearchModalComponent.module.scss';

interface Props {
  isLoading: boolean;
  workspaceMap: Map<Workspace, Project[]>;
  onModalClose: () => void;
}

const WorkspaceQuickSearchModalComponent: React.FC<Props> = ({
  isLoading,
  workspaceMap,
  onModalClose,
}: Props) => {
  const [searchText, setSearchText] = useState<string>('');

  const onChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchText(e.target.value);
  }, []);

  const onHideModal = useCallback(() => {
    setSearchText('');
    onModalClose();
  }, [onModalClose]);

  const onClickProject = useCallback(
    (project: Project) => {
      routeToReactUrl(paths.projectDetails(project.id));
      onHideModal();
    },
    [onHideModal],
  );

  const onClickWorkspace = useCallback(
    (workspaceId: number) => {
      routeToReactUrl(paths.workspaceDetails(workspaceId));
      onHideModal();
    },
    [onHideModal],
  );

  const getNodesForProject = useCallback(
    (projects: Project[], text: string) => {
      const treeChildren: TreeDataNode[] = projects
        .filter((project) => project.name.toLocaleLowerCase().includes(text))
        .map((project) => ({
          key: `project-${project.id}`,
          title: (
            <div className={`${css.flexRow} ${css.ellipsis}`}>
              <Icon decorative name="project" size="small" />
              <Link onClick={() => onClickProject(project)}>{project.name}</Link>
              <span>({project.numExperiments})</span>
            </div>
          ),
        }));
      return treeChildren;
    },
    [onClickProject],
  );

  const treeData: TreeDataNode[] = useMemo(() => {
    const text = searchText.toLocaleLowerCase();
    const data: TreeDataNode[] = Array.from(workspaceMap)
      .map(([workspace, projects]) => {
        const isWorkspaceNameIncluded = workspace.name.toLocaleLowerCase().includes(text);
        const children = getNodesForProject(projects, text);
        return {
          children: children,
          isWorkspaceIncluded:
            searchText.length > 0 ? isWorkspaceNameIncluded || children.length > 0 : true,
          key: `workspace-${workspace.id}`,
          title: (
            <div className={`${css.flexRow} ${css.ellipsis}`}>
              <Icon name="workspaces" title="Workspace" />
              <Link onClick={() => onClickWorkspace(workspace.id)}>{workspace.name}</Link>
            </div>
          ),
        };
      })
      .filter((item) => item.isWorkspaceIncluded);
    return data;
  }, [getNodesForProject, onClickWorkspace, searchText, workspaceMap]);

  return (
    <Modal
      cancel={false}
      size="large"
      submit={{
        handleError: () => {},
        handler: onHideModal,
        text: 'Close',
      }}
      title="Workspace Quick Search"
      onClose={onHideModal}>
      <Input
        autoFocus
        placeholder="Search workspace or project"
        prefix={<Icon name="search" title="Search" />}
        value={searchText}
        onChange={onChange}
      />
      <div className={css.modalBody}>
        {isLoading ? (
          <Spinner center spinning tip={'Loading...'} />
        ) : (
          <>
            {treeData.length === 0 ? (
              <Message icon="warning" title="No matching workspace or projects" />
            ) : (
              <Tree defaultExpandAll treeData={treeData} />
            )}
          </>
        )}
      </div>
    </Modal>
  );
};

export default WorkspaceQuickSearchModalComponent;
