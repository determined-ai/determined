import { Tabs } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useHistory, useLocation, useParams } from 'react-router';

import Page from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import { useStore } from 'contexts/Store';
import useFeature from 'hooks/useFeature';
import { useFetchUsers } from 'hooks/useFetch';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import { paths } from 'routes/utils';
import { getWorkspace, getGroups } from 'services/api';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import { isNotFound } from 'shared/utils/service';
import { Workspace } from 'types';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import css from './WorkspaceDetails.module.scss';
import WorkspaceDetailsHeader from './WorkspaceDetails/WorkspaceDetailsHeader';
import WorkspaceMembers from './WorkspaceDetails/WorkspaceMembers';
import WorkspaceProjects from './WorkspaceDetails/WorkspaceProjects';
import { isEqual } from 'shared/utils/data';
import handleError from 'utils/error';

interface Params {
  tab: string;
  workspaceId: string;
}

export enum WorkspaceDetailsTab {
  Members = 'members',
  Projects = 'projects',
}

const WorkspaceDetails: React.FC = () => {
  const rbacEnabled = useFeature().isOn('rbac');
  const { workspaceId } = useParams<Params>();
  const [workspace, setWorkspace] = useState<Workspace>();
  const [groups, setGroups] = useState<V1GroupSearchResult[]>();
  const [pageError, setPageError] = useState<Error>();
  const [canceler] = useState(new AbortController());
  const [tabKey, setTabKey] = useState<WorkspaceDetailsTab>(WorkspaceDetailsTab.Projects);
  const pageRef = useRef<HTMLElement>(null);
  const id = parseInt(workspaceId);
  const history = useHistory();
  const location = useLocation();
  const basePath = paths.workspaceDetails(workspaceId);
  const { canViewWorkspace } = usePermissions();

  const fetchWorkspace = useCallback(async () => {
    try {
      const response = await getWorkspace({ id }, { signal: canceler.signal });
      setWorkspace(response);
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    }
  }, [canceler.signal, id, pageError]);

  const fetchUsers = useFetchUsers(canceler);

  const fetchGroups = useCallback(async (): Promise<void> => {
    try {
      const response = await getGroups({}, { signal: canceler.signal });

      setGroups((prev) => {
        if (isEqual(prev, response.groups)) return prev;
        return response.groups || [];
      });
    } catch (e) {
      handleError(e);
    }
  }, [canceler.signal]);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([fetchWorkspace(), fetchUsers(), fetchGroups()]);
  }, [fetchWorkspace, fetchUsers, fetchGroups]);

  usePolling(fetchAll, { rerunOnNewFn: true });

  const handleTabChange = useCallback(
    (activeTab) => {
      const tab = activeTab as WorkspaceDetailsTab;
      history.replace(`${basePath}/${tab}`);
      setTabKey(tab);
    },
    [basePath, history],
  );

  useEffect(() => {
    // Set the correct pathname to ensure
    // that user settings will save.

    if (
      !location.pathname.includes(WorkspaceDetailsTab.Projects) &&
      !location.pathname.includes(WorkspaceDetailsTab.Members)
    )
      history.replace(`${basePath}/${tabKey}`);
  }, [basePath, history, location.pathname, tabKey]);

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  if (isNaN(id)) {
    return <Message title={`Invalid Workspace ID ${workspaceId}`} />;
  } else if (pageError) {
    if (isNotFound(pageError)) return <PageNotFound />;
    const message = `Unable to fetch Workspace ${workspaceId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!workspace) {
    return <Spinner tip={`Loading workspace ${workspaceId} details...`} />;
  }

  if (!canViewWorkspace({ workspace: { id } })) {
    return <PageNotFound />;
  }

  const groupsList = groups?.map(group => group.group) || [] 
  return (
    <Page
      className={css.base}
      containerRef={pageRef}
      headerComponent={<WorkspaceDetailsHeader groups={groupsList} fetchWorkspace={fetchAll} workspace={workspace} />}
      id="workspaceDetails">
      {rbacEnabled ? (
        <Tabs activeKey={tabKey} destroyInactiveTabPane onChange={handleTabChange}>
          <Tabs.TabPane destroyInactiveTabPane key={WorkspaceDetailsTab.Projects} tab="Projects">
            <WorkspaceProjects id={id} pageRef={pageRef} workspace={workspace} />
          </Tabs.TabPane>
          <Tabs.TabPane destroyInactiveTabPane key={WorkspaceDetailsTab.Members} tab="Members">
            <WorkspaceMembers pageRef={pageRef} workspace={workspace} groups={groupsList}/>
          </Tabs.TabPane>
        </Tabs>
      ) : (
        <WorkspaceProjects id={id} pageRef={pageRef} workspace={workspace} />
      )}
    </Page>
  );
};

export default WorkspaceDetails;
