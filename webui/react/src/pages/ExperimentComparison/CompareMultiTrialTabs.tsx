import { Tabs } from 'antd';
import React from 'react';
import { useParams } from 'react-router';

import Spinner from 'shared/components/Spinner/Spinner';
import { ExperimentBase, ExperimentVisualizationType } from 'types';

const { TabPane } = Tabs;

interface Params {
  ids: string;
  viz?: ExperimentVisualizationType;
}

const CompareVisualization = React.lazy(() => {
  return import('./CompareVisualization');
});

export interface Props {
  experiments: ExperimentBase[];
  fetchExperimentDetails: () => void;
}

const CompareMultiTrialTabs: React.FC<Props> = (
  { experiments }: Props,
) => {
  const { viz } = useParams<Params>();

  return (
    <Tabs className="no-padding" defaultActiveKey="visualization">
      <TabPane key="visualization" tab="Visualization">
        <React.Suspense fallback={<Spinner tip="Loading experiment visualization..." />}>
          <CompareVisualization
            basePath="experiment-comparison"
            experiments={experiments}
            type={viz}
          />
        </React.Suspense>
      </TabPane>

    </Tabs>
  );
};

export default CompareMultiTrialTabs;
