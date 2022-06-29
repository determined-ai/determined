import React from 'react';

import Spinner from 'shared/components/Spinner/Spinner';

const CompareVisualization = React.lazy(() => {
  return import('./CompareVisualization');
});

const ExperimentComparisonTabs: React.FC = () => {
  return (

    <React.Suspense fallback={<Spinner tip="Loading experiment visualization..." />}>
      <CompareVisualization />
    </React.Suspense>

  );
};

export default ExperimentComparisonTabs;
