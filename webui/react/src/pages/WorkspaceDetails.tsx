import { Tabs } from 'antd';
import React, { useCallback, useRef, useState } from 'react';
import { useParams } from 'react-router';
import Page from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import WorkspaceProjects from './WorkspaceProjects';
import { useStore } from 'contexts/Store';
import { useFetchUsers } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import { paths } from 'routes/utils';
import { getWorkspace} from 'services/api';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import { isNotFound} from 'shared/utils/service';
import {Workspace } from 'types';
import css from './WorkspaceDetails.module.scss';
import WorkspaceDetailsHeader from './WorkspaceDetails/WorkspaceDetailsHeader';
import WorkspaceMembers from './WorkspaceMembers';
import { routeToReactUrl } from 'shared/utils/routes';

interface Params {
  workspaceId: string;
  tab: string;
}

export enum WorkspaceDetailsTab {
  Projects = 'projects',
  Members = 'members'
}


const WorkspaceDetails: React.FC = () => {
  const { users, auth: { user } } = useStore();
  const { workspaceId, tab } = useParams<Params>();
  const [ workspace, setWorkspace ] = useState<Workspace>();
  const [ pageError, setPageError ] = useState<Error>();
  const [ canceler ] = useState(new AbortController());
  const tabKey = tab ? tab: WorkspaceDetailsTab.Projects;
  const pageRef = useRef<HTMLElement>(null);
  const basePath = paths.workspaceDetails(workspaceId);
  const id = parseInt(workspaceId);

  const fetchWorkspace = useCallback(async () => {
    try {
      const response = await getWorkspace({ id }, { signal: canceler.signal });
      setWorkspace(response);
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    }
  }, [ canceler.signal, id, pageError ]);

  const fetchUsers = useFetchUsers(canceler);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([ fetchWorkspace(), fetchUsers() ]);
  }, [ fetchWorkspace, ]);

  usePolling(fetchAll, { rerunOnNewFn: true });

  const handleTabChange = useCallback(() => {
    const activeKey = tabKey as WorkspaceDetailsTab;
    if(activeKey == WorkspaceDetailsTab.Projects){
      routeToReactUrl(paths.workspaceMembers(workspaceId))
    } else {
      routeToReactUrl(paths.workspaceProjects(workspaceId))
    }
  }, [ basePath, history, tabKey ]);


  if (isNaN(id)) {
    return <Message title={`Invalid Workspace ID ${workspaceId}`} />;
  } else if (pageError) {
    if (isNotFound(pageError)) return <PageNotFound />;
    const message =
      `Unable to fetch Workspace ${workspaceId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!workspace) {
    return <Spinner tip={`Loading workspace ${workspaceId} details...`} />;
  }

  return (
    <Page
      className={css.base}
      containerRef={pageRef}
      headerComponent={(
        <WorkspaceDetailsHeader
          fetchWorkspace={fetchAll}
          workspace={workspace}
        />
      )}
      id="workspaceDetails">
      <Tabs
        activeKey={tabKey}
        onChange={handleTabChange}
        destroyInactiveTabPane
        >
        <Tabs.TabPane
          key={WorkspaceDetailsTab.Projects}
          tab="Projects"
          destroyInactiveTabPane
          >
        <WorkspaceProjects workspace={workspace} id={id} pageRef={pageRef}/>
      </Tabs.TabPane>
      <Tabs.TabPane
          key={WorkspaceDetailsTab.Members}
          tab="Members"
          destroyInactiveTabPane
          >
           <WorkspaceMembers users={users} workspace={workspace} pageRef={pageRef}/>
      </Tabs.TabPane>
      </Tabs>
    </Page>
  );
};

export default WorkspaceDetails;
