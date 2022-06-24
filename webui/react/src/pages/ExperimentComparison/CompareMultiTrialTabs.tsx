import { Tabs } from 'antd';
import React, { useCallback, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import ExperimentTrials from 'pages/ExperimentDetails/ExperimentTrials';
import { paths } from 'routes/utils';
import Spinner from 'shared/components/Spinner/Spinner';
import { ExperimentBase, ExperimentVisualizationType } from 'types';

const { TabPane } = Tabs;

enum TabType {
  Configuration = 'configuration',
  Trials = 'trials',
  Visualization = 'visualization',
  Notes = 'notes',
}

interface Params {
  ids: string;
  viz?: ExperimentVisualizationType;
}

const TAB_KEYS = Object.values(TabType);
const DEFAULT_TAB_KEY = TabType.Visualization;

const CompareVisualization = React.lazy(() => {
  return import('./CompareVisualization');
});

export interface Props {
  experiments: ExperimentBase[];
  fetchExperimentDetails: () => void;
  pageRef: React.RefObject<HTMLElement>;
}

const CompareMultiTrialTabs: React.FC<Props> = (
  { experiments, pageRef }: Props,
) => {
  const { viz, ids } = useParams<Params>();

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
