import { Loadable } from 'hew/utils/loadable';
import _ from 'lodash';
import { useObservable } from 'micro-observables';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';

import Page, { BreadCrumbRoute } from 'components/Page';
import { terminalRunStates } from 'constants/states';
import usePolling from 'hooks/usePolling';
import { paths } from 'routes/utils';
import { getExperimentDetails } from 'services/api';
import workspaceStore from 'stores/workspaces';
import { ExperimentBase, Workspace } from 'types';
import { isAborted } from 'utils/service';

type Params = {
  searchId: string;
};

const SearchDetails: React.FC = () => {
  const { searchId } = useParams<Params>();
  const [experiment, setExperiment] = useState<ExperimentBase>();
  const [pageError, setPageError] = useState<Error>();
  const canceler = useRef<AbortController>();
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));

  const fetchExperimentDetails = useCallback(async () => {
    if (!searchId) return;
    try {
      const newExperiment = await getExperimentDetails(
        { id: parseInt(searchId) },
        { signal: canceler.current?.signal },
      );
      setExperiment((prevExperiment) =>
        _.isEqual(prevExperiment, newExperiment) ? prevExperiment : newExperiment,
      );
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    }
  }, [pageError, searchId]);

  const { stopPolling } = usePolling(fetchExperimentDetails, { rerunOnNewFn: true });

  useEffect(() => {
    if (experiment && terminalRunStates.has(experiment.state)) {
      stopPolling();
    }
  }, [experiment, stopPolling]);

  const workspaceName = workspaces.find((ws: Workspace) => ws.id === experiment?.workspaceId)?.name;

  const pageBreadcrumb: BreadCrumbRoute[] = [
    workspaceName && experiment?.workspaceId !== 1
      ? {
        breadcrumbName: workspaceName,
        path: paths.workspaceDetails(experiment?.workspaceId ?? 1),
      }
      : {
        breadcrumbName: 'Uncategorized Experiments',
        path: paths.projectDetails(1),
      },
  ];

  return (
    <Page
      breadcrumb={pageBreadcrumb}
      // containerRef={pageRef}
      headerComponent={
        searchId
      }
      // notFound={pageError && isNotFound(pageError)}
      stickyHeader
      title={`Search ${searchId}`}
    />
  );

};

export default SearchDetails;
