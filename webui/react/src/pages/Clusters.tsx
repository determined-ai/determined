import { Tabs } from 'antd';
import React, { useCallback, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import Page from 'components/Page';
import { paths } from 'routes/utils';

import ClusterLogs from './ClusterLogs';

const { TabPane } = Tabs;

enum TabType {
  Overview = 'overview',
  HistoricalUsage = 'historical-usage',
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

  const handleTabChange = useCallback(key => {
    setTabKey(key);
    history.replace(key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`);
  }, [ basePath, history ]);

  return (
    <Page bodyNoPadding id="cluster" stickyHeader title="Cluster">
      <Tabs className="no-padding" defaultActiveKey={tabKey} onChange={handleTabChange}>
        <TabPane key="logs" tab="Cluster Logs">
          <ClusterLogs />
        </TabPane>
      </Tabs>
    </Page>
  );
};

export default Clusters;
