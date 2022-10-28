import { Tabs } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Page from 'components/Page';
import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';
import { ValueOf } from 'shared/types';
import { useAgents, useClusterOverview } from 'stores/agents';
import { Loadable } from 'utils/loadable';

import ClusterHistoricalUsage from './Cluster/ClusterHistoricalUsage';
import ClusterLogs from './ClusterLogs';
import css from './Clusters.module.scss';
import ClustersOverview, { clusterStatusText } from './Clusters/ClustersOverview';

const { TabPane } = Tabs;

const TabType = {
  HistoricalUsage: 'historical-usage',
  Logs: 'logs',
  Overview: 'overview',
} as const;

type TabType = ValueOf<typeof TabType>;

type Params = {
  tab?: TabType;
};

const DEFAULT_TAB_KEY = TabType.Overview;

const Clusters: React.FC = () => {
  const { tab } = useParams<Params>();
  const basePath = paths.clusters();
  const navigate = useNavigate();

  const [tabKey, setTabKey] = useState<TabType>(tab || DEFAULT_TAB_KEY);
  const { resourcePools } = useStore();
  const overview = useClusterOverview();
  const agents = useAgents();

  const cluster = useMemo(() => {
    return Loadable.match(Loadable.all([agents, overview]), {
      Loaded: ([agents, overview]) => clusterStatusText(overview, resourcePools, agents),
      NotLoaded: () => undefined, // TODO show spinner when this is loading
    });
  }, [overview, resourcePools, agents]);

  const handleTabChange = useCallback(
    (key) => {
      setTabKey(key);
      navigate(key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`, { replace: true });
    },
    [basePath, navigate],
  );

  return (
    <Page bodyNoPadding id="cluster" title={`Cluster ${cluster ? `- ${cluster}` : ''}`}>
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
