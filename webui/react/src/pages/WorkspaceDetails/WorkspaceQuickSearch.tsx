import { Button, Input, Modal, Tree } from 'antd';
import type { DefaultOptionType } from 'rc-tree-select/lib/TreeSelect';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Link from 'components/Link';
import { paths } from 'routes/utils';
import { getWorkspaceProjects, getWorkspaces } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { Project, Workspace } from 'types';
import handleError from 'utils/error';

import css from './WorkspaceQuickSearch.module.scss';

const WorkspaceQuickSearch: React.FC = () => {
  const [ searchText, setSearchText ] = useState<string>('');
  const [ workspaces, setWorkspaces ] = useState<Workspace[]>([]);
  const [ projects, setProjects ] = useState<Project[][]>([]);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ isModalVisible, setIsModalVisible ] = useState(false);
  const canceler = useRef(new AbortController());

  const onChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchText(e.target.value);
  };

  const onShowModal = () => setIsModalVisible(true);

  const onHideModal = () => setIsModalVisible(false);

  const fetchData = useCallback(async () => {
    try {
      const workspaceRes = await getWorkspaces(
        { limit: 0, sortBy: 'SORT_BY_NAME' },
        { signal: canceler.current.signal },
      );
      const filteredWorkspaces = workspaceRes.workspaces.filter((w) => !w.immutable);
      const projectapi = filteredWorkspaces
        .map((workspace) =>
          getWorkspaceProjects(
            { id: workspace.id, sortBy: 'SORT_BY_NAME' },
            { signal: canceler.current.signal },
          ));
      const projectRes = (await Promise.all(projectapi)).map((project) => project.projects);
      setWorkspaces(filteredWorkspaces);
      setProjects(projectRes);
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

  useEffect(() => {
    fetchData();
  }, [ fetchData ]);

  const treeData = useMemo(() => {
    const map: Map<number, DefaultOptionType[]> = new Map();

    for (const workspace of workspaces) {
      if (workspace.name.includes(searchText)) {
        map.set(workspace.id, []);
      }
    }
    for (const project of projects) {
      for (const p of project) {
        if (p.name.includes(searchText)) {
          const tempArr = [];
          if (map.has(p.workspaceId)) {
            tempArr.push(...(map.get(p.workspaceId) as DefaultOptionType[]));
          }
          tempArr.push({
            title: (
              <div className={`${css.flexRow} ${css.ellipsis}`}>
                <Icon name="experiment" />
                <Link onClick={() => routeToReactUrl(paths.projectDetails(p.id))}>
                  {p.name}
                </Link>
              </div>
            ),
            value: `project-${p.name}`,
          });
          map.set(p.workspaceId, tempArr);
        }
      }
    }

    const arr: DefaultOptionType[] = Array.from(map).map(([ k, v ]) => (
      {
        children: v,
        title: (
          <div className={`${css.flexRow} ${css.ellipsis}`}>
            <Icon name="workspaces" />
            <Link onClick={() => {
              routeToReactUrl(paths.workspaceDetails(k));
            }}>{workspaces.find((workspace) => workspace.id === k)?.name}
            </Link>
          </div>
        ),
        value: `workspace-${k}`,
      }
    ));
    return arr;
  }, [ projects, searchText, workspaces ]);

  if (isLoading) {
    return <div>loading</div>;
  }

  return (
    <div>
      <Button ghost type="primary" onClick={onShowModal}><Icon name="search" /></Button>
      <Modal
        closable={false}
        footer={null}
        title={(
          <Input
            placeholder="Search and Jump to the workspace or project"
            prefix={<Icon name="search" />}
            width={'100%'}
            onChange={onChange}
          />
        )}
        visible={isModalVisible}
        width={'50vw'}
        onCancel={onHideModal}>
        <div className={css.modalBody}>
          {treeData.length === 0 ? <div>Empty</div> : (
            <Tree
              defaultExpandAll
              selectable={false}
              treeData={treeData}
            />
          )}
        </div>
      </Modal>
    </div>
  );
};

export default WorkspaceQuickSearch;
