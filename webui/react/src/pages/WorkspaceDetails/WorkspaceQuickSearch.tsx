import { Input, Modal, Tree } from 'antd';
import type { DefaultOptionType } from 'rc-tree-select/lib/TreeSelect';
import React, { useCallback, useMemo, useRef, useState } from 'react';

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
  const canceler = useRef(new AbortController());

  const fetchData = useCallback(async () => {
    try {
      const workspaceResponse = await getWorkspaces(
        { limit: 0, sortBy: 'SORT_BY_NAME' },
        { signal: canceler.current.signal },
      );
      const filteredWorkspaces = workspaceResponse.workspaces.filter((w) => !w.immutable);
      const projectAPIList = filteredWorkspaces
        .map((workspace) =>
          getWorkspaceProjects(
            { id: workspace.id, sortBy: 'SORT_BY_NAME' },
            { signal: canceler.current.signal },
          ));
      const projectResponse = (await Promise.all(projectAPIList))
        .map((project) => project.projects);

      // Promise.all preserves the order
      const tempWorkspaceMap: Map<Workspace, Project[]> = new Map();
      for (const workspace of filteredWorkspaces.reverse()) {
        const projects = projectResponse.pop();
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

  const onChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchText(e.target.value);
  };

  const onShowModal = () => {
    setIsModalVisible(true);
    fetchData();
  };

  const onHideModal = () => setIsModalVisible(false);

  const onClickProject = useCallback((project: Project) => {
    routeToReactUrl(paths.projectDetails(project.id));
    onHideModal();
  }, []);

  const onClickWorkspace = useCallback((workspaceId: number) => {
    routeToReactUrl(paths.workspaceDetails(workspaceId));
    onHideModal();
  }, []);

  const treeData: DefaultOptionType[] = useMemo(() => {
    const data: DefaultOptionType[] = Array.from(workspaceMap)
      .map(([ workspace, projects ]) => {
        const treeChildren: DefaultOptionType[] = projects
          .filter((project) => project.name.includes(searchText))
          .map((project) => ({
            title: (
              <div className={`${css.flexRow} ${css.ellipsis}`}>
                <Icon name="experiment" />
                <Link onClick={() => onClickProject(project)}>{project.name}</Link>
              </div>
            ),
            value: `project-${project.id}`,
          }));
        return ({
          children: treeChildren,
          title: (
            <div className={`${css.flexRow} ${css.ellipsis}`}>
              <Icon name="workspaces" />
              <Link onClick={() => onClickWorkspace(workspace.id)}>{workspace.name}</Link>
            </div>
          ),
          value: `workspace-${workspace.id}`,
        });
      })
      .filter((item) => searchText.length > 0 ? item.children.length > 0 : true);
    return data;
  }, [ onClickProject, onClickWorkspace, searchText, workspaceMap ]);

  return (
    <div>
      <div onClick={onShowModal}>
        {children}
      </div>
      <Modal
        closable={false}
        footer={null}
        title={(
          <Input
            placeholder="Search and Jump to workspace or project"
            prefix={<Icon name="search" />}
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
                  <Message title="No data found" type={MessageType.Empty} /> : (
                    <Tree defaultExpandAll selectable={false} treeData={treeData} />
                  )}
              </>
            )}
        </div>
      </Modal>
    </div>
  );
};

export default WorkspaceQuickSearch;
