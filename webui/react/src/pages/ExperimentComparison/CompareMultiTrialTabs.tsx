import { Tabs } from 'antd';
import React from 'react';

import Spinner from 'shared/components/Spinner/Spinner';

const { TabPane } = Tabs;

const CompareVisualization = React.lazy(() => {
  return import('./CompareVisualization');
});

const CompareMultiTrialTabs: React.FC = () => {
  return (
    <Tabs className="no-padding" defaultActiveKey="visualization">
      <TabPane key="visualization" tab="Visualization">
        <React.Suspense fallback={<Spinner tip="Loading experiment visualization..." />}>
          <CompareVisualization />
        </React.Suspense>
      </TabPane>

    </Tabs>
  );
};

export default CompareMultiTrialTabs;
