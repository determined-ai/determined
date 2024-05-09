import Pivot, { PivotProps } from 'hew/Pivot';
import React, { useCallback, useMemo, useState } from 'react';
import Joyride from 'react-joyride';
import { useNavigate, useParams } from 'react-router-dom';

import Page from 'components/Page';
import usePermissions from 'hooks/usePermissions';
import ClusterOverview from 'pages/Cluster/ClusterOverview';
import { paths } from 'routes/utils';
import clusterStore from 'stores/cluster';
import determinedStore from 'stores/determinedInfo';
import { ValueOf } from 'types';
import { useObservable } from 'utils/observable';

import ClusterHistoricalUsage from './Cluster/ClusterHistoricalUsage';
import css from './Cluster.module.scss';
import ClusterLogs from './ClusterLogs';

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

const Cluster: React.FC = () => {
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

  const steps = useMemo(
    () => [
      {
        content: 'This is my awesome feature!',
        // disableBeacon: true,
        target: '[data-node-key="historical-usage"]',
      },
      {
        content: 'This is another awesome feature!',
        //disableBeacon: true,
        target: '[data-node-key="logs"]',
      },
      // {
      //   content: 'Going back!',
      //   disableBeacon: true,
      //   target: '[data-node-key="checkpoints"]',
      // },
    ],
    [],
  );

  const tabItems: PivotProps['items'] = useMemo(() => {
    type Unboxed<T> = T extends (infer U)[] ? U : T;
    type TabType = Unboxed<PivotProps['items']>;

    const clusterOverview: Readonly<TabType> = {
      children: <ClusterOverview />,
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
    const tabs: PivotProps['items'] = [];

    if (rbacEnabled) {
      tabs.push(clusterOverview);
      if (canAdministrateUsers) {
        tabs.push(historicalUsage, masterLogs);
      }
    } else {
      // if RBAC is not enabled, show all tabs
      tabs.push(clusterOverview, historicalUsage, masterLogs);
    }

    return tabs;
  }, [canAdministrateUsers, rbacEnabled]);

  return (
    <>
      <Joyride showProgress showSkipButton steps={steps} />
      <Page
        breadcrumb={[
          {
            breadcrumbName: 'Cluster',
            path: paths.clusters(),
          },
        ]}
        id="cluster"
        title={`Cluster ${clusterStatus ? `- ${clusterStatus}` : ''}`}>
        <div className={css.pivoter}>
          <Pivot defaultActiveKey={tabKey} items={tabItems} onChange={handleTabChange} />
        </div>
      </Page>
    </>
  );
};

export default Cluster;
