import { Tabs } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import Spinner from 'components/Spinner';
import ExperimentTrials from 'pages/ExperimentDetails/ExperimentTrials';
import { paths } from 'routes/utils';
import { ExperimentBase, ExperimentVisualizationType } from 'types';

const { TabPane } = Tabs;

enum TabType {
  Configuration = 'configuration',
  Trials = 'trials',
  Visualization = 'visualization',
}

interface Params {
  tab?: TabType;
  viz?: ExperimentVisualizationType;
}

const TAB_KEYS = Object.values(TabType);
const DEFAULT_TAB_KEY = TabType.Visualization;

const ExperimentConfiguration = React.lazy(() => {
  return import('./ExperimentConfiguration');
});
const ExperimentVisualization = React.lazy(() => {
  return import('./ExperimentVisualization');
});

export interface Props {
  experiment: ExperimentBase;
}

const ExperimentMultiTrialTabs: React.FC<Props> = (
  { experiment }: Props,
) => {
  const { tab, viz } = useParams<Params>();
  const history = useHistory();
  const defaultTabKey = tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY;
  const [ tabKey, setTabKey ] = useState(defaultTabKey);

  const basePath = paths.experimentDetails(experiment.id);

  const handleTabChange = useCallback(key => {
    setTabKey(key);
    history.replace(`${basePath}/${key}`);
  }, [ basePath, history ]);

  // Sets the default sub route.
  useEffect(() => {
    if (!tab || (tab && !TAB_KEYS.includes(tab))) {
      history.replace(`${basePath}/${tabKey}`);
    }
  }, [ basePath, history, tab, tabKey ]);

  return (
    <Tabs className="no-padding" defaultActiveKey={tabKey} onChange={handleTabChange}>
      <TabPane key="visualization" tab="Visualization">
        <React.Suspense fallback={<Spinner tip="Loading experiment visualization..." />}>
          <ExperimentVisualization
            basePath={`${basePath}/${TabType.Visualization}`}
            experiment={experiment}
            type={viz} />
        </React.Suspense>
      </TabPane>
      <TabPane key="trials" tab="Trials">
        <ExperimentTrials experiment={experiment} />
      </TabPane>
      <TabPane key="configuration" tab="Configuration">
        <React.Suspense fallback={<Spinner tip="Loading text editor..." />}>
          <ExperimentConfiguration experiment={experiment} />
        </React.Suspense>
      </TabPane>
    </Tabs>
  );
};

export default ExperimentMultiTrialTabs;
