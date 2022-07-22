import queryString from 'query-string';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useLocation } from 'react-router';

import Page from 'components/Page';
import Message from 'shared/components/Message';

import ComparisonHeader from './ExperimentComparison/CompareHeader';
import ComparisonTabs from './ExperimentComparison/CompareTabs';
interface Query {
  id?: string[];
}

const ExperimentComparison: React.FC = () => {
  const location = useLocation();

  const experimentIds: number[] = useMemo(() => {
    const query: Query = queryString.parse(location.search);
    if (query.id && typeof query.id === 'string'){
      return [ parseInt(query.id) ];
    } else if (Array.isArray(query.id)) {

      return query.id.map((x) => parseInt(x));
    }
    return [];
  }, [ location.search ]);

  const [ canceler ] = useState(new AbortController());

  const pageRef = useRef<HTMLElement>(null);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  if (!experimentIds || !experimentIds.length) {
    return <Message title="No Experiments chosen for comparison" />;
  }

  return (
    <Page
      bodyNoPadding
      containerRef={pageRef}
      headerComponent={(
        <ComparisonHeader />
      )}
      stickyHeader
      title="Compare Experiments">
      <ComparisonTabs />
    </Page>
  );
};

export default ExperimentComparison;
