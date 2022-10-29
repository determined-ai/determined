import { Tabs } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Active from 'components/ExperimentIcons/Active';
import Queue from 'components/ExperimentIcons/Queue';
import Spinner from 'components/ExperimentIcons/Spinner';
import Page from 'components/Page';
import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';
import { ValueOf } from 'shared/types';

import ClusterHistoricalUsage from './Cluster/ClusterHistoricalUsage';
import ClusterLogs from './ClusterLogs';
import css from './Clusters.module.scss';
import ClustersOverview, { clusterStatusText } from './Clusters/ClustersOverview';

import ExperimentIcons from 'components/ExperimentIcons'
import { RunState } from 'types';

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
  const { agents, cluster: overview, resourcePools } = useStore();

  const cluster = useMemo(() => {
    return clusterStatusText(overview, resourcePools, agents);
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
      <ExperimentIcons state={RunState.Queued} /> <ExperimentIcons state={RunState.Queued} />
      <Spinner type="bowtie" /><Spinner type="bowtie" /> <Spinner type="half" /><Spinner type="half" />
      <Queue /><Queue />
      {/* <Tabs className="no-padding" defaultActiveKey={tabKey} onChange={handleTabChange}>
        <TabPane key="overview" tab="Overview">
          <ClustersOverview />
        </TabPane>
        <TabPane className={css.tab} key="historical-usage" tab="Historical Usage">
          <ClusterHistoricalUsage />
        </TabPane>
        <TabPane key="logs" tab="Master Logs">
          <ClusterLogs />
        </TabPane>
      </Tabs> */}
    </Page>
  );
};

export default Clusters;
