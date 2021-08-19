import React, { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router';

import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import { useStore } from 'contexts/Store';
import useCreateExperimentModal, { CreateExperimentType } from 'hooks/useCreateExperimentModal';
import usePolling from 'hooks/usePolling';
import ExperimentDetailsHeader from 'pages/ExperimentDetails/ExperimentDetailsHeader';
import {
  getExperimentDetails, getExpValidationHistory, isNotFound,
} from 'services/api';
import { isAborted } from 'services/utils';
import { ExperimentBase, TrialDetails, ValidationHistory } from 'types';
import { isEqual } from 'utils/data';
import { isSingleTrialExperiment } from 'utils/experiment';
import { terminalRunStates } from 'utils/types';

import ExperimentMultiTrialTabs from './ExperimentDetails/ExperimentMultiTrialTabs';
import ExperimentSingleTrialTabs from './ExperimentDetails/ExperimentSingleTrialTabs';

interface Params {
  experimentId: string;
}

const ExperimentDetails: React.FC = () => {
  const { experimentId } = useParams<Params>();
  const { auth: { user } } = useStore();
  const [ canceler ] = useState(new AbortController());
  const [ experiment, setExperiment ] = useState<ExperimentBase>();
  const [ trial, setTrial ] = useState<TrialDetails>();
  const [ valHistory, setValHistory ] = useState<ValidationHistory[]>([]);
  const [ pageError, setPageError ] = useState<Error>();
  const [ isSingleTrial, setIsSingleTrial ] = useState<boolean>();

  const id = parseInt(experimentId);

  const { showModal } = useCreateExperimentModal();

  const fetchExperimentDetails = useCallback(async () => {
    try {
      const [ experimentData, validationHistory ] = await Promise.all([
        getExperimentDetails({ id }, { signal: canceler.signal }),
        getExpValidationHistory({ id }, { signal: canceler.signal }),
      ]);
      if (!isEqual(experimentData, experiment)) setExperiment(experimentData);
      if (!isEqual(validationHistory, valHistory)) setValHistory(validationHistory);
      setIsSingleTrial(
        isSingleTrialExperiment(experimentData),
      );
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e);
    }
  }, [
    experiment,
    id,
    canceler.signal,
    pageError,
    valHistory,
  ]);

  const { stopPolling } = usePolling(fetchExperimentDetails);

  const showForkModal = useCallback((): void => {
    if (!experiment) return;
    showModal({ experiment, type: CreateExperimentType.Fork });
  }, [ experiment, showModal ]);

  const showContinueTrial = useCallback((): void => {
    if (!experiment || !trial) return;
    showModal({ experiment, trial, type: CreateExperimentType.ContinueTrial });
  }, [ experiment, showModal, trial ]);

  const handleSingleTrialLoad = useCallback((trial: TrialDetails) => {
    setTrial(trial);
  }, []);

  useEffect(() => {
    if (experiment && terminalRunStates.has(experiment.state)) {
      stopPolling();
    }
  }, [ experiment, stopPolling ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  if (isNaN(id)) {
    return <Message title={`Invalid Experiment ID ${experimentId}`} />;
  } else if (pageError) {
    const message = isNotFound(pageError) ?
      `Unable to find Experiment ${experimentId}` :
      `Unable to fetch Experiment ${experimentId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!experiment || isSingleTrial === undefined) {
    return <Spinner tip={`Loading experiment ${experimentId} details...`} />;
  }

  return (
    <Page
      bodyNoPadding
      headerComponent={<ExperimentDetailsHeader
        curUser={user}
        experiment={experiment}
        fetchExperimentDetails={fetchExperimentDetails}
        showContinueTrial={trial ? showContinueTrial : undefined}
        showForkModal={showForkModal}
        trial={trial}
      />}
      stickyHeader
      title={`Experiment ${experimentId}`}>
      {isSingleTrial ? (
        <ExperimentSingleTrialTabs experiment={experiment} onTrialLoad={handleSingleTrialLoad} />
      ) : (
        <ExperimentMultiTrialTabs experiment={experiment} />
      )}
    </Page>
  );
};

export default ExperimentDetails;
