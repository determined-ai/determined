import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';

import Page from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import { terminalRunStates } from 'constants/states';
import ExperimentDetailsHeader from 'pages/ExperimentDetails/ExperimentDetailsHeader';
import ExperimentMultiTrialTabs from 'pages/ExperimentDetails/ExperimentMultiTrialTabs';
import ExperimentSingleTrialTabs from 'pages/ExperimentDetails/ExperimentSingleTrialTabs';
import { TrialInfoBoxMultiTrial } from 'pages/TrialDetails/TrialInfoBox';
import { getExperimentDetails } from 'services/api';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner/Spinner';
import usePolling from 'shared/hooks/usePolling';
import { isEqual } from 'shared/utils/data';
import { isNotFound } from 'shared/utils/service';
import { isAborted } from 'shared/utils/service';
import { ExperimentBase, TrialItem } from 'types';
import { isSingleTrialExperiment } from 'utils/experiment';

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
  } else if (pageError) {
    if (isNotFound(pageError)) return <PageNotFound />;
    const message = `${ERROR_MESSAGE} ${experimentId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!experiment || isSingleTrial === undefined) {
    return <Spinner tip={`Loading experiment ${experimentId} details...`} />;
  }

  return (
    <Page
      containerRef={pageRef}
      headerComponent={
        <ExperimentDetailsHeader
          experiment={experiment}
          fetchExperimentDetails={fetchExperimentDetails}
          trial={trial}
        />
      }
      stickyHeader
      title={`Experiment ${experimentId}`}>
      {isSingleTrial ? (
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
      )}
    </Page>
  );
};

export default ExperimentDetails;
