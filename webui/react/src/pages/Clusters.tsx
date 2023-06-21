import type { TabsProps } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Pivot from 'components/kit/Pivot';
import Page from 'components/Page';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import clusterStore from 'stores/cluster';
import determinedStore from 'stores/determinedInfo';
import { ValueOf } from 'types';
import { useObservable } from 'utils/observable';

import ClusterHistoricalUsage from './Cluster/ClusterHistoricalUsage';
import ClusterLogs from './ClusterLogs';
import ClustersOverview from './Clusters/ClustersOverview';

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
  const { rbacEnabled } = useObservable(determinedStore.info);
  const { canAdministrateUsers } = usePermissions();
  const { tab } = useParams<Params>();
  const basePath = paths.clusters();
  const navigate = useNavigate();

  const [tabKey, setTabKey] = useState<TabType>(tab || DEFAULT_TAB_KEY);

  const clusterStatus = useObservable(clusterStore.clusterStatus);

  const handleTabChange = useCallback(
    (key: string) => {
      setTabKey(key as TabType);
      navigate(key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`, { replace: true });
    },
    [basePath, navigate],
  );

  const tabItems: TabsProps['items'] = useMemo(() => {
    type Unboxed<T> = T extends (infer U)[] ? U : T;
    type TabType = Unboxed<TabsProps['items']>;

    const clustersOverview: Readonly<TabType> = {
      children: <ClustersOverview />,
      key: TabType.Overview,
      label: 'Overview',
    };
    const historicalUsage: Readonly<TabType> = {
      children: <ClusterHistoricalUsage />,
      key: TabType.HistoricalUsage,
      label: 'Historical Usage',
    };
    const masterLogs: Readonly<TabType> = {
      children: <ClusterLogs />,
      key: TabType.Logs,
      label: 'Master Logs',
    };
    const tabs: TabsProps['items'] = [];

    if (rbacEnabled) {
      tabs.push(clustersOverview);
      if (canAdministrateUsers) {
        tabs.push(historicalUsage, masterLogs);
      }
    } else {
      // if RBAC is not enabled, show all tabs
      tabs.push(clustersOverview, historicalUsage, masterLogs);
    }

    return tabs;
  }, [canAdministrateUsers, rbacEnabled]);

  return (
    <Page
      breadcrumb={[
        {
          breadcrumbName: 'Cluster',
          path: paths.clusters(),
        },
      ]}
      id="cluster"
      title={`Cluster ${clusterStatus ? `- ${clusterStatus}` : ''}`}>
      <Pivot defaultActiveKey={tabKey} items={tabItems} onChange={handleTabChange} />
    </Page>
  );
};

export default Clusters;
