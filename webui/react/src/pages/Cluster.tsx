import { Tabs } from 'antd';
import React, { useCallback, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import Page from 'components/Page';
import { paths } from 'routes/utils';

import ClusterHistoricalUsage from './Cluster/ClusterHistoricalUsage';
import ClusterOverview from './Cluster/ClusterOverview';

const { TabPane } = Tabs;

enum TabType {
  Overview = 'overview',
  HistoricalUsage = 'historical-usage',
}

interface Params {
  tab?: TabType;
}

const DEFAULT_TAB_KEY = TabType.Overview;

const Cluster: React.FC = () => {
  const { tab } = useParams<Params>();
  const basePath = paths.cluster();
  const history = useHistory();

  const [ tabKey, setTabKey ] = useState<TabType>(tab || DEFAULT_TAB_KEY);

  const handleTabChange = useCallback(key => {
    setTabKey(key);
    history.replace(key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`);
  }, [ basePath, history ]);

  return (
    <Page id="cluster" stickyHeader title="Cluster">
      <Tabs defaultActiveKey={tabKey} onChange={handleTabChange}>
        <TabPane key="overview" tab="Overview">
          <ClusterOverview />
        </TabPane>
        <TabPane key="historical-usage" tab="Historical Usage">
          <ClusterHistoricalUsage />
        </TabPane>
      </Tabs>
    </Page>
  );
};

export default Cluster;
