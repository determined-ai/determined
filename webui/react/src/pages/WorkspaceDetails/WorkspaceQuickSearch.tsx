import { useModal } from 'hew/Modal';
import { Loadable } from 'hew/utils/loadable';
import React, { useCallback, useEffect, useState } from 'react';

import { getWorkspaceProjects } from 'services/api';
import workspaceStore from 'stores/workspaces';
import { Project, Workspace } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { useObservable } from 'utils/observable';

import WorkspaceQuickSearchModalComponent from './WorkspaceQuickSearchModalComponent';

interface Props {
  children: React.ReactNode;
}

const WorkspaceQuickSearch: React.FC<Props> = ({ children }: Props) => {
  const [workspaceMap, setWorkspaceMap] = useState<Map<Workspace, Project[]>>(new Map());
  const [isLoading, setIsLoading] = useState(true);
  const [isModalVisible, setIsModalVisible] = useState(false);
  const WorkspaceQuickSearchModal = useModal(WorkspaceQuickSearchModalComponent);
  const workspaceObservable = useObservable(workspaceStore.mutables);
  const workspaces = Loadable.getOrElse([], workspaceObservable);

  useEffect(() => {
    if (isModalVisible) WorkspaceQuickSearchModal.open();
  }, [isModalVisible, WorkspaceQuickSearchModal]);

  const fetchData = useCallback(async () => {
    try {
      const projectResponse = await getWorkspaceProjects({ id: 0, sortBy: 'SORT_BY_NAME' });

      const projectMap = new Map<number, Project[]>();
      for (const project of projectResponse.projects) {
        projectMap.set(project.workspaceId, [
          ...(projectMap.get(project.workspaceId) ?? []),
          project,
        ]);
      }

      const tempWorkspaceMap: Map<Workspace, Project[]> = new Map();
      for (const workspace of workspaces) {
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
  }, [workspaces]);

  const onShowModal = useCallback(() => {
    fetchData();
    setIsModalVisible(true);
  }, [fetchData]);

  return (
    <>
      <div onClick={onShowModal}>{children}</div>
      <WorkspaceQuickSearchModal.Component
        isLoading={isLoading}
        workspaceMap={workspaceMap}
        onModalClose={() => setIsModalVisible(false)}
      />
    </>
  );
};

export default WorkspaceQuickSearch;
