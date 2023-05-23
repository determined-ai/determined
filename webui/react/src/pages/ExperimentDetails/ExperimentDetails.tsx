import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';

import Page, { BreadCrumbRoute } from 'components/Page';
import { terminalRunStates } from 'constants/states';
import ExperimentDetailsHeader from 'pages/ExperimentDetails/ExperimentDetailsHeader';
import ExperimentMultiTrialTabs from 'pages/ExperimentDetails/ExperimentMultiTrialTabs';
import ExperimentSingleTrialTabs from 'pages/ExperimentDetails/ExperimentSingleTrialTabs';
import { TrialInfoBoxMultiTrial } from 'pages/TrialDetails/TrialInfoBox';
import { paths } from 'routes/utils';
import { getExperimentDetails } from 'services/api';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner/Spinner';
import usePolling from 'shared/hooks/usePolling';
import { isEqual } from 'shared/utils/data';
import { isNotFound } from 'shared/utils/service';
import { isAborted } from 'shared/utils/service';
import workspaceStore from 'stores/workspaces';
import { ExperimentBase, TrialItem, Workspace } from 'types';
import { isSingleTrialExperiment } from 'utils/experiment';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

type Params = {
  experimentId: string;
};

export const INVALID_ID_MESSAGE = 'Invalid Experiment ID';
export const ERROR_MESSAGE = 'Unable to fetch Experiment';

const ExperimentDetails: React.FC = () => {
  const { experimentId } = useParams<Params>();
  const [experiment, setExperiment] = useState<ExperimentBase>();
  const [trial, setTrial] = useState<TrialItem>();
  const [pageError, setPageError] = useState<Error>();
  const [isSingleTrial, setIsSingleTrial] = useState<boolean>();
  const pageRef = useRef<HTMLElement>(null);
  const canceler = useRef<AbortController>();
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));
  const id = parseInt(experimentId ?? '');

  const fetchExperimentDetails = useCallback(async () => {
    try {
      const newExperiment = await getExperimentDetails(
        { id },
        { signal: canceler.current?.signal },
      );
      setExperiment((prevExperiment) =>
        isEqual(prevExperiment, newExperiment) ? prevExperiment : newExperiment,
      );
      setIsSingleTrial(isSingleTrialExperiment(newExperiment));
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    }
  }, [id, pageError]);

  const { stopPolling } = usePolling(fetchExperimentDetails, { rerunOnNewFn: true });

  const handleSingleTrialUpdate = useCallback((trial: TrialItem) => {
    setTrial(trial);
  }, []);

  useEffect(() => {
    if (experiment && terminalRunStates.has(experiment.state)) {
      stopPolling();
    }
  }, [experiment, stopPolling]);

  useEffect(() => {
    fetchExperimentDetails();
  }, [fetchExperimentDetails]);

  useEffect(() => {
    canceler.current = new AbortController();
    return () => {
      canceler.current?.abort();
      canceler.current = undefined;
    };
  }, []);

  if (isNaN(id)) {
    return <Message title={`${INVALID_ID_MESSAGE} ${experimentId}`} />;
  } else if (pageError && !isNotFound(pageError)) {
    const message = `${ERROR_MESSAGE} ${experimentId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!pageError && (!experiment || isSingleTrial === undefined)) {
    return <Spinner tip={`Loading experiment ${experimentId} details...`} />;
  }

  const workspaceName = workspaces.find((ws: Workspace) => ws.id === experiment?.workspaceId)?.name;

  const pageBreadcrumb: BreadCrumbRoute[] = [
    {
      breadcrumbName:
        workspaceName && experiment?.workspaceId !== 1
          ? workspaceName
          : 'Uncategorized Experiments',
      path: paths.workspaceDetails(experiment?.workspaceId ?? 1),
    },
  ];

  if (experiment?.projectName && experiment?.projectId && experiment?.projectId !== 1)
    pageBreadcrumb.push({
      breadcrumbName: experiment?.projectName ?? '',
      path: paths.projectDetails(experiment?.projectId),
    });

  pageBreadcrumb.push({
    breadcrumbName: experiment?.name ?? '',
    path: paths.experimentDetails(id),
  });

  return (
    <Page
      breadcrumb={pageBreadcrumb}
      containerRef={pageRef}
      headerComponent={
        experiment && (
          <ExperimentDetailsHeader
            experiment={experiment}
            fetchExperimentDetails={fetchExperimentDetails}
            trial={trial}
          />
        )
      }
      notFound={pageError && isNotFound(pageError)}
      stickyHeader
      title={`Experiment ${experimentId}`}>
      {experiment &&
        (isSingleTrial ? (
          <ExperimentSingleTrialTabs
            experiment={experiment}
            fetchExperimentDetails={fetchExperimentDetails}
            pageRef={pageRef}
            onTrialUpdate={handleSingleTrialUpdate}
          />
        ) : (
          <>
            <TrialInfoBoxMultiTrial experiment={experiment} />
            <ExperimentMultiTrialTabs
              experiment={experiment}
              fetchExperimentDetails={fetchExperimentDetails}
              pageRef={pageRef}
            />
          </>
        ))}
    </Page>
  );
};

export default ExperimentDetails;
