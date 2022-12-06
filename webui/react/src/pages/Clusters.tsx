import { Tabs } from 'antd';
import type { TabsProps } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Page from 'components/Page';
import { paths } from 'routes/utils';
import { ValueOf } from 'shared/types';
import { useAgents, useClusterOverview } from 'stores/agents';
import { useResourcePools } from 'stores/resourcePools';
import { Loadable } from 'utils/loadable';

import ClusterHistoricalUsage from './Cluster/ClusterHistoricalUsage';
import ClusterLogs from './ClusterLogs';
import css from './Clusters.module.scss';
import ClustersOverview, { clusterStatusText } from './Clusters/ClustersOverview';

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
  const loadableResourcePools = useResourcePools();
  const resourcePools = Loadable.getOrElse([], loadableResourcePools); // TODO show spinner when this is loading
  const overview = useClusterOverview();
  const agents = useAgents();

  const cluster = useMemo(() => {
    return Loadable.match(Loadable.all([agents, overview]), {
      Loaded: ([agents, overview]) => clusterStatusText(overview, resourcePools, agents),
      NotLoaded: () => undefined, // TODO show spinner when this is loading
    });
  }, [overview, resourcePools, agents]);

  const handleTabChange = useCallback(
    (key: string) => {
      setTabKey(key as TabType);
      navigate(key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`, { replace: true });
    },
    [basePath, navigate],
  );

  const tabItems: TabsProps['items'] = useMemo(() => {
    return [
      { children: <ClustersOverview />, key: TabType.Overview, label: 'Overview' },
      {
        children: (
          <div className={css.tab}>
            <ClusterHistoricalUsage />
          </div>
        ),
        key: TabType.HistoricalUsage,
        label: 'Historical Usage',
      },
      {
        children: <ClusterLogs />,
        key: TabType.Logs,
        label: 'Master Logs',
      },
    ];
  }, []);

  return (
    <Page bodyNoPadding id="cluster" title={`Cluster ${cluster ? `- ${cluster}` : ''}`}>
      <Tabs
        className="no-padding"
        defaultActiveKey={tabKey}
        items={tabItems}
        onChange={handleTabChange}
      />
    </Page>
  );
};

export default Clusters;
