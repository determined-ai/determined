import { ProjectOutlined } from '@ant-design/icons';
import { Input, Modal, Tree } from 'antd';
import type { DefaultOptionType } from 'rc-tree-select/lib/TreeSelect';
import React, { useCallback, useMemo, useState } from 'react';

import Link from 'components/Link';
import { paths } from 'routes/utils';
import { getWorkspaceProjects, getWorkspaces } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { Project, Workspace } from 'types';
import handleError from 'utils/error';

import css from './WorkspaceQuickSearch.module.scss';

interface Props {
  children: React.ReactChild;
}

const WorkspaceQuickSearch: React.FC<Props> = ({ children }: Props) => {
  const [ searchText, setSearchText ] = useState<string>('');
  const [ workspaceMap, setWorkspaceMap ] = useState<Map<Workspace, Project[]>>(new Map());
  const [ isLoading, setIsLoading ] = useState(true);
  const [ isModalVisible, setIsModalVisible ] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      const workspaceResponse = await getWorkspaces({ limit: 0, sortBy: 'SORT_BY_NAME' });
      const filteredWorkspaces = workspaceResponse.workspaces.filter((w) => !w.immutable);
      const projectResponse = await getWorkspaceProjects({ id: 0, sortBy: 'SORT_BY_NAME' });

      const projectMap = new Map<number, Project[]>();
      for (const project of projectResponse.projects) {
        projectMap.set(
          project.workspaceId,
          [ ...projectMap.get(project.workspaceId) ?? [], project ],
        );
      }

      const tempWorkspaceMap: Map<Workspace, Project[]> = new Map();
      for (const workspace of filteredWorkspaces) {
        const projects = projectMap.get(workspace.id);
        tempWorkspaceMap.set(workspace, projects ?? []);
      }
      setWorkspaceMap(tempWorkspaceMap);
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to fetch data.',
        silent: false,
        type: ErrorType.Api,
      });
    } finally {
      setIsLoading(false);
    }
  }, [ ]);

  const onChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchText(e.target.value);
  }, []);

  const onShowModal = useCallback(() => {
    setIsModalVisible(true);
    fetchData();
  }, [ fetchData ]);

  const onHideModal = useCallback(() => {
    setIsModalVisible(false);
    setSearchText('');
  }, []);

  const onClickProject = useCallback((project: Project) => {
    routeToReactUrl(paths.projectDetails(project.id));
    onHideModal();
  }, [ onHideModal ]);

  const onClickWorkspace = useCallback((workspaceId: number) => {
    routeToReactUrl(paths.workspaceDetails(workspaceId));
    onHideModal();
  }, [ onHideModal ]);

  const getNodesForProject = useCallback((projects: Project[], text: string) => {
    const treeChildren: DefaultOptionType[] = projects
      .filter((project) => project.name.toLocaleLowerCase().includes(text))
      .map((project) => ({
        title: (
          <div className={`${css.flexRow} ${css.ellipsis}`}>
            <ProjectOutlined style={{ fontSize: '16px' }} />
            <Link onClick={() => onClickProject(project)}>{project.name}</Link>
          </div>
        ),
        value: `project-${project.id}`,
      }));
    return treeChildren;
  }, [ onClickProject ]);

  const treeData: DefaultOptionType[] = useMemo(() => {
    const text = searchText.toLocaleLowerCase();
    const data: DefaultOptionType[] = Array.from(workspaceMap)
      .map(([ workspace, projects ]) => {
        const isWorkspaceNameIncluded = workspace.name.toLocaleLowerCase().includes(text);
        const children = getNodesForProject(projects, text);
        return ({
          children: children,
          isWorkspaceIncluded: searchText.length > 0 ?
            (isWorkspaceNameIncluded || children.length > 0) : true,
          title: (
            <div className={`${css.flexRow} ${css.ellipsis}`}>
              <Icon name="workspaces" />
              <Link onClick={() => onClickWorkspace(workspace.id)}>{workspace.name}</Link>
            </div>
          ),
          value: `workspace-${workspace.id}`,
        });
      })
      .filter((item) => item.isWorkspaceIncluded);
    return data;
  }, [ getNodesForProject, onClickWorkspace, searchText, workspaceMap ]);

  return (
    <>
      <div onClick={onShowModal}>{children}</div>
      <Modal
        closable={false}
        footer={null}
        title={(
          <Input
            autoFocus
            placeholder="Search workspace or project"
            prefix={<Icon name="search" />}
            value={searchText}
            width={'100%'}
            onChange={onChange}
          />
        )}
        visible={isModalVisible}
        width={'clamp(520px, 50vw, 1000px)'}
        onCancel={onHideModal}>
        <div className={css.modalBody}>
          {isLoading ?
            <Spinner center tip={'Loading...'} /> : (
              <>
                {treeData.length === 0 ?
                  <Message title="No matching workspace or projects" type={MessageType.Empty} /> : (
                    <Tree defaultExpandAll selectable={false} treeData={treeData} />
                  )}
              </>
            )}
        </div>
      </Modal>
    </>
  );
};

export default WorkspaceQuickSearch;
