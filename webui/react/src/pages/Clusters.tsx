import { Tabs } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import Page from 'components/Page';
import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';

import ClusterHistoricalUsage from './Cluster/ClusterHistoricalUsage';
import ClusterLogs from './ClusterLogs';
import css from './Clusters.module.scss';
import ClustersOverview, { clusterStatusText } from './Clusters/ClustersOverview';

const { TabPane } = Tabs;

enum TabType {
  Overview = 'overview',
  HistoricalUsage = 'historical-usage',
  Logs = 'logs'
}

interface Params {
  tab?: TabType;
}

const DEFAULT_TAB_KEY = TabType.Overview;

const Clusters: React.FC = () => {
  const { tab } = useParams<Params>();
  const basePath = paths.clusters();
  const history = useHistory();

  const [ tabKey, setTabKey ] = useState<TabType>(tab || DEFAULT_TAB_KEY);
  const { agents, cluster: overview, resourcePools } = useStore();

  const cluster = useMemo(() => {
    return clusterStatusText(overview, resourcePools, agents);
  }, [ overview, resourcePools, agents ]);

  const handleTabChange = useCallback(key => {
    setTabKey(key);
    history.replace(key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`);
  }, [ basePath, history ]);

  return (
    <Page
      bodyNoPadding
      id="cluster"
      stickyHeader
      title={`Cluster ${cluster ? `- ${cluster}` : ''}`}>
      <Tabs className="no-padding" defaultActiveKey={tabKey} onChange={handleTabChange}>
        <TabPane key="overview" tab="Overview">
          <ClustersOverview />
        </TabPane>
        <TabPane className={css.tab} key="historical-usage" tab="Historical Usage">
          <ClusterHistoricalUsage />
        </TabPane>
        <TabPane key="logs" tab="Master Logs">
          <ClusterLogs />
        </TabPane>
      </Tabs>
    </Page>
  );
};

export default Clusters;
