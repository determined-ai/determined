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

// const TAB_KEYS = Object.values(TabType);
// const DEFAULT_TAB_KEY = TabType.Visualization;

const ExperimentVisualization = React.lazy(() => {
  return import('./CompareVisualization');
});

export interface Props {
  experiments: ExperimentBase[];
  fetchExperimentDetails: () => void;
  pageRef: React.RefObject<HTMLElement>;
}

const ExperimentMultiTrialTabs: React.FC<Props> = (
  { experiments }: Props,
) => {
  const { viz } = useParams<Params>();
  return (

    <React.Suspense fallback={<Spinner tip="Loading experiment visualization..." />}>
      <ExperimentVisualization
        basePath="experiment-comparison"
        experiments={experiments}
        type={viz}
      />
    </React.Suspense>

  );
};

export default ExperimentMultiTrialTabs;
