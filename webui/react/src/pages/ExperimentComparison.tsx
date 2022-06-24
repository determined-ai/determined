import queryString from 'query-string';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation } from 'react-router';

import Page from 'components/Page';
import {
  getExperimentDetails,
} from 'services/api';
import Message, { MessageType } from 'shared/components/Message';
import { isEqual } from 'shared/utils/data';
import { ExperimentBase } from 'types';

import { isAborted } from '../shared/utils/service';

import ComparisonHeader from './ExperimentComparison/CompareHeader';
import ComparisonTabs from './ExperimentComparison/CompareTabs';
interface Query {
  id?: string[];
}

const ExperimentComparison: React.FC = () => {
  const location = useLocation();

  const experimentIds = useMemo(() => {
    const query: Query = queryString.parse(location.search);
    if(query.id && typeof query.id === 'string'){
      return [query.id]
    }
    return query.id ?? [];
  }, [ location.search ]);

  const [ canceler ] = useState(new AbortController());
  const [ experiments, setExperiments ] = useState<ExperimentBase[]>([]);
  const [ pageError, setPageError ] = useState<Error>();
  const pageRef = useRef<HTMLElement>(null);

  const fetchExperimentDetails = useCallback(async () => {
    try {
      const experimentsData = await Promise.all(
        experimentIds.map((id) =>
          getExperimentDetails({ id: parseInt(id) }, { signal: canceler.signal })),
      );
      if (!isEqual(experimentsData, experiments)) setExperiments(experimentsData);
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    }
  }, [
    experiments,
    experimentIds,
    canceler.signal,
    pageError,
  ]);

  useEffect(() => {
    fetchExperimentDetails();
  }, []);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  if (!experimentIds || !experimentIds.length) {
    return <Message title="No Experiments chosen for comparison" />;
  } else if (pageError) {
    return <Message title="Unable to compare experiments" type={MessageType.Warning} />;
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
      <ComparisonTabs
        experiments={experiments}
        fetchExperimentDetails={fetchExperimentDetails}
        pageRef={pageRef}
      />
    </Page>
  );
};

export default ExperimentComparison;
