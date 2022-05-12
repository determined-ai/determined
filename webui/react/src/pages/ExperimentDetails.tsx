import React, { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router';

import Page from 'components/Page';
import Spinner from 'components/Spinner';
import { terminalRunStates } from 'constants/states';
import { useStore } from 'contexts/Store';
import usePolling from 'hooks/usePolling';
import ExperimentDetailsHeader from 'pages/ExperimentDetails/ExperimentDetailsHeader';
import {
  getExperimentDetails, getExpValidationHistory, isNotFound,
} from 'services/api';
import { getProject } from 'services/api';
import { isAborted } from 'services/utils';
import Message, { MessageType } from 'shared/components/message';
import { ExperimentBase, Project, TrialDetails, ValidationHistory, Workspace } from 'types';
import { isEqual } from 'utils/data';
import { isSingleTrialExperiment } from 'utils/experiment';

import ExperimentMultiTrialTabs from './ExperimentDetails/ExperimentMultiTrialTabs';
import ExperimentSingleTrialTabs from './ExperimentDetails/ExperimentSingleTrialTabs';

interface Params {
  experimentId: string;
  projectId: string;
  workspaceId: string;
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

  const[ workspace, setWorkspace ] = useState<Workspace>();
  const [ project, setProject ] = useState<Project>();

  const id = parseInt(experimentId);

  const fetchProject = useCallback(async () => {
    if (!experiment?.projectId) return;
    try {
      const response = await getProject({ id: experiment?.projectId }, { signal: canceler.signal });
      setProject(prev => {
        if (isEqual(prev, response)) return prev;
        return response;
      });
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    }
  }, [ canceler.signal, experiment?.projectId, pageError ]);

  const fetchExperimentDetails = useCallback(async () => {
    try {
      const [ newExperiment, newValHistory ] = await Promise.all([
        getExperimentDetails({ id }, { signal: canceler.signal }),
        getExpValidationHistory({ id }, { signal: canceler.signal }),
      ]);
      setExperiment((prevExperiment) =>
        isEqual(prevExperiment, newExperiment) ? prevExperiment : newExperiment);
      setValHistory((prevValHistory) =>
        isEqual(prevValHistory, newValHistory) ? prevValHistory : newValHistory);
      setIsSingleTrial(
        isSingleTrialExperiment(newExperiment),
      );
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    }
  }, [
    id,
    canceler.signal,
    pageError,
  ]);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([ fetchProject(), fetchExperimentDetails() ]);
  }, [ fetchProject, fetchExperimentDetails ]);

  const { stopPolling } = usePolling(fetchAll);

  const handleSingleTrialLoad = useCallback((trial: TrialDetails) => {
    setTrial(trial);
  }, []);

  useEffect(() => {
    if (experiment && terminalRunStates.has(experiment.state)) {
      stopPolling();
    }
  }, [ experiment, stopPolling ]);

  useEffect(() => {
    fetchExperimentDetails();
  }, [ fetchExperimentDetails ]);

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
      headerComponent={(
        <ExperimentDetailsHeader
          curUser={user}
          experiment={experiment}
          fetchExperimentDetails={fetchExperimentDetails}
          project={project}
          trial={trial}
          workspace={workspace}
        />
      )}
      stickyHeader
      title={`Experiment ${experimentId}`}>
      {isSingleTrial ? (
        <ExperimentSingleTrialTabs
          experiment={experiment}
          fetchExperimentDetails={fetchExperimentDetails}
          onTrialLoad={handleSingleTrialLoad}
        />
      ) : (
        <ExperimentMultiTrialTabs
          experiment={experiment}
          fetchExperimentDetails={fetchExperimentDetails}
        />
      )}
    </Page>
  );
};

export default ExperimentDetails;
