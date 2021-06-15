import { Tabs } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import Spinner from 'components/Spinner';
import { ExperimentTabsProps } from 'pages/ExperimentDetails';
import { paths } from 'routes/utils';

const { TabPane } = Tabs;

enum TabType {
  Configuration = 'configuration',
  Hyperparameters = 'hyperparameters',
  Logs = 'logs',
  Overview = 'overview',
  Profiler = 'profiler',
  Workloads = 'workloads',
}

interface Params {
  tab?: TabType;
}

const TAB_KEYS = Object.values(TabType);
const DEFAULT_TAB_KEY = TabType.Overview;

const ExperimentConfiguration = React.lazy(() => {
  return import('./ExperimentConfiguration');
});

const ExperimentSingleTrialTabs: React.FC<ExperimentTabsProps> = (
  { experiment }: ExperimentTabsProps,
) => {
  const { tab } = useParams<Params>();
  const history = useHistory();
  const defaultTabKey = tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY;
  const [ tabKey, setTabKey ] = useState(defaultTabKey);

  const basePath = paths.experimentDetails(experiment.id);

  const handleTabChange = useCallback(key => {
    setTabKey(key);
    history.replace(key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`);
  }, [ basePath, history ]);

  useEffect(() => {
    if (tab && (!TAB_KEYS.includes(tab) || tab === DEFAULT_TAB_KEY)) {
      history.replace(basePath);
    }
  }, [ basePath, history, tab ]);

  return (
    <Tabs defaultActiveKey={tabKey} onChange={handleTabChange}>
      <TabPane key="overview" tab="Overview">
        Overview
      </TabPane>
      <TabPane key="hyperparameters" tab="Hyperparameters">
        Hyperparameters
      </TabPane>
      <TabPane key="workloads" tab="Workloads">
        Workloads
      </TabPane>
      <TabPane key="configuration" tab="Configuration">
        <React.Suspense fallback={<Spinner />}>
          <ExperimentConfiguration experiment={experiment} />
        </React.Suspense>
      </TabPane>
      <TabPane key="profiler" tab="Profiler">
        Profiler
      </TabPane>
      <TabPane key="logs" tab="Logs">
        Logs
      </TabPane>
    </Tabs>
  );
};

export default ExperimentSingleTrialTabs;
