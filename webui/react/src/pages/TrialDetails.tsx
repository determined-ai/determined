import { Tabs } from 'antd';
import axios from 'axios';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import ContinueTrial, { ContinueTrialHandles } from 'components/ContinueTrial';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import TrialDetailsHeader, { Action as TrialAction } from 'pages/TrialDetails/TrialDetailsHeader';
import TrialDetailsHyperparameters from 'pages/TrialDetails/TrialDetailsHyperparameters';
import TrialDetailsLogs from 'pages/TrialDetails/TrialDetailsLogs';
import TrialDetailsOverview from 'pages/TrialDetails/TrialDetailsOverview';
import TrialDetailsProfiles from 'pages/TrialDetails/TrialDetailsProfiles';
import TrialDetailsWorkloads from 'pages/TrialDetails/TrialDetailsWorkloads';
import TrialRangeHyperparameters from 'pages/TrialDetails/TrialRangeHyperparameters';
import { paths } from 'routes/utils';
import { getExperimentDetails, getTrialDetails, isNotFound } from 'services/api';
import { ApiState } from 'services/types';
import { isAborted } from 'services/utils';
import { ExperimentBase, TrialDetails } from 'types';
import { isSingleTrialExperiment } from 'utils/experiment';
import { terminalRunStates } from 'utils/types';

const { TabPane } = Tabs;

enum TabType {
  Hyperparameters = 'hyperparameters',
  Logs = 'logs',
  Overview = 'overview',
  Profiler = 'profiler',
  Workloads = 'workloads',
}

interface Params {
  experimentId?: string;
  tab?: TabType;
  trialId: string;
}

const DEFAULT_TAB_KEY = TabType.Overview;

const TrialDetailsComp: React.FC = () => {
  const [ canceler ] = useState(new AbortController());
  const [ experiment, setExperiment ] = useState<ExperimentBase>();
  const [ source ] = useState(axios.CancelToken.source());
  const continueTrialRef = useRef<ContinueTrialHandles>(null);
  const history = useHistory();
  const routeParams = useParams<Params>();

  const [ tabKey, setTabKey ] = useState<TabType>(routeParams.tab || DEFAULT_TAB_KEY);
  const [ trialDetails, setTrialDetails ] = useState<ApiState<TrialDetails>>({
    data: undefined,
    error: undefined,
    isLoading: true,
    source,
  });
  const basePath = paths.trialDetails(routeParams.trialId, routeParams.experimentId);
  const trialId = parseInt(routeParams.trialId);

  const trial = trialDetails.data;

  const fetchExperimentDetails = useCallback(async () => {
    if (!trial) return;

    try {
      const response = await getExperimentDetails(
        { id: trial.experimentId },
        { signal: canceler.signal },
      );
      setExperiment(response);

      // Experiment id does not exist in route, reroute to the one with it
      if (!routeParams.experimentId) {
        history.replace(paths.trialDetails(trial.id, trial.experimentId));
      }
    } catch (e) {
      if (axios.isCancel(e)) return;
      handleError({
        error: e,
        message: 'Failed to load experiment details.',
        publicMessage: 'Failed to load experiment details.',
        publicSubject: 'Unable to fetch Trial Experiment Detail',
        silent: false,
        type: ErrorType.Api,
      });
    }
  }, [
    canceler,
    history,
    routeParams.experimentId,
    trial,
  ]);

  const fetchTrialDetails = useCallback(async () => {
    try {
      const response = await getTrialDetails({ id: trialId }, { signal: canceler.signal });
      setTrialDetails(prev => ({ ...prev, data: response, isLoading: false }));
    } catch (e) {
      if (!trialDetails.error && !isAborted(e)) {
        setTrialDetails(prev => ({ ...prev, error: e }));
      }
    }
  }, [ canceler, trialDetails.error, trialId ]);

  const handleActionClick = useCallback((action: TrialAction) => {
    switch (action) {
      case TrialAction.Continue:
        continueTrialRef.current?.show();
        break;
    }
  }, [ continueTrialRef ]);

  const handleTabChange = useCallback(key => {
    setTabKey(key);
    history.replace(key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`);
  }, [ basePath, history ]);

  const { stopPolling } = usePolling(fetchTrialDetails);

  useEffect(() => {
    fetchExperimentDetails();
  }, [ fetchExperimentDetails ]);

  useEffect(() => {
    if (trialDetails.data && terminalRunStates.has(trialDetails.data.state)) {
      stopPolling();
    }
  }, [ trialDetails.data, stopPolling ]);

  useEffect(() => {
    return () => {
      source.cancel();
      canceler.abort();
    };
  }, [ canceler, source ]);

  if (isNaN(trialId)) {
    return <Message title={`Invalid Trial ID ${routeParams.trialId}`} />;
  }

  if (trialDetails.error !== undefined) {
    const message = isNotFound(trialDetails.error) ?
      `Unable to find Trial ${trialId}` :
      `Unable to fetch Trial ${trialId}`;
    return <Message
      message={trialDetails.error.message}
      title={message}
      type={MessageType.Warning} />;
  }

  if (!trial || !experiment) {
    return <Spinner tip={`Fetching ${trial ? 'experiment' : 'trial'} information...`} />;
  }

  return (
    <Page
      headerComponent={<TrialDetailsHeader
        experiment={experiment}
        fetchTrialDetails={fetchTrialDetails}
        handleActionClick={handleActionClick}
        trial={trial}
      />}
      stickyHeader
      title={`Trial ${trialId}`}
    >
      <Tabs defaultActiveKey={tabKey} onChange={handleTabChange}>
        <TabPane key={TabType.Overview} tab="Overview">
          <TrialDetailsOverview experiment={experiment} trial={trial} />
        </TabPane>
        <TabPane key={TabType.Hyperparameters} tab="Hyperparameters">
          {
            isSingleTrialExperiment(experiment) ?
              <TrialDetailsHyperparameters experiment={experiment} trial={trial} /> :
              <TrialRangeHyperparameters experiment={experiment} trial={trial} />
          }
        </TabPane>
        <TabPane key={TabType.Workloads} tab="Workloads">
          <TrialDetailsWorkloads experiment={experiment} trial={trial} />
        </TabPane>
        <TabPane key={TabType.Profiler} tab="Profiler">
          <TrialDetailsProfiles experiment={experiment} trial={trial} />
        </TabPane>
        <TabPane key={TabType.Logs} tab="Logs">
          <TrialDetailsLogs experiment={experiment} trial={trial} />
        </TabPane>
      </Tabs>
      <ContinueTrial experiment={experiment} ref={continueTrialRef} trial={trial} />
    </Page>
  );
};

export default TrialDetailsComp;
