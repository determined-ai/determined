import { Alert, Tabs } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import Spinner from 'components/Spinner';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import usePrevious from 'hooks/usePrevious';
import { paths } from 'routes/utils';
import { getExpTrials, getTrialDetails } from 'services/api';
import { ExperimentBase, RunState, TrialDetails } from 'types';
import { terminalRunStates } from 'utils/types';

import TrialDetailsHyperparameters from '../TrialDetails/TrialDetailsHyperparameters';
import TrialDetailsLogs from '../TrialDetails/TrialDetailsLogs';
import TrialDetailsOverview from '../TrialDetails/TrialDetailsOverview';
import TrialDetailsProfiles from '../TrialDetails/TrialDetailsProfiles';
import TrialDetailsWorkloads from '../TrialDetails/TrialDetailsWorkloads';

const { TabPane } = Tabs;

enum TabType {
  Configuration = 'configuration',
  Hyperparameters = 'hyperparameters',
  Logs = 'logs',
  Overview = 'overview',
  Profiler = 'profiler',
  Workloads = 'workloads',
}

interface Params {
  tab?: TabType;
}

const TAB_KEYS = Object.values(TabType);
const DEFAULT_TAB_KEY = TabType.Overview;

const ExperimentConfiguration = React.lazy(() => {
  return import('./ExperimentConfiguration');
});

export interface Props {
  experiment: ExperimentBase;
  onTrialLoad?: (trial: TrialDetails) => void;
}

const NoDataAlert = <Alert message="No data available." type="warning" />;

const ExperimentSingleTrialTabs: React.FC<Props> = ({ experiment, onTrialLoad }: Props) => {
  const history = useHistory();
  const [ trialId, setFirstTrialId ] = useState<number>();
  const [ wontHaveTrials, setWontHaveTrials ] = useState<boolean>(false);
  const prevTrialId = usePrevious(trialId, undefined);
  const { tab } = useParams<Params>();
  const [ canceler ] = useState(new AbortController());
  const [ trialDetails, setTrialDetails ] = useState<TrialDetails>();
  const [ hasLoaded, setHasLoaded ] = useState(false);

  const basePath = paths.experimentDetails(experiment.id);
  const defaultTabKey = tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY;

  const [ tabKey, setTabKey ] = useState(defaultTabKey);

  const handleTabChange = useCallback(key => {
    setTabKey(key);
    history.replace(`${basePath}/${key}`);
  }, [ basePath, history ]);

  // Sets the default sub route.
  useEffect(() => {
    if (!tab || (tab && !TAB_KEYS.includes(tab))) {
      history.replace(`${basePath}/${tabKey}`);
    }
  }, [ basePath, history, tab, tabKey ]);

  const fetchFirstTrialId = useCallback(async () => {
    try {
      // make sure the experiment is in terminal state before the request is made.
      const isTerminalExp = terminalRunStates.has(experiment.state);
      const expTrials = await getExpTrials(
        { id: experiment.id, limit: 2 },
        { signal: canceler.signal },
      );
      const firstTrial = expTrials.trials[0];
      if (firstTrial) {
        if (onTrialLoad) onTrialLoad(firstTrial);
        setFirstTrialId(firstTrial.id);
      } else if (isTerminalExp) {
        setWontHaveTrials(true);
      }
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: 'Failed to fetch experiment trials.',
        silent: true,
        type: ErrorType.Server,
      });
    }
  }, [ canceler, experiment.id, experiment.state, onTrialLoad ]);

  const fetchTrialDetails = useCallback(async () => {
    if (!trialId) return;

    try {
      const response = await getTrialDetails({ id: trialId }, { signal: canceler.signal });
      setTrialDetails(response);
      setHasLoaded(true);
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: 'Failed to fetch experiment trials.',
        silent: true,
        type: ErrorType.Server,
      });
    }
  }, [ canceler, trialId ]);

  const { stopPolling } = usePolling(fetchTrialDetails);
  const { stopPolling: stopPollingFirstTrialId } = usePolling(fetchFirstTrialId);

  useEffect(() => {
    if (trialDetails && terminalRunStates.has(trialDetails.state)) {
      stopPolling();
    }
  }, [ trialDetails, stopPolling ]);

  useEffect(() => {
    if (wontHaveTrials || trialId !== undefined) stopPollingFirstTrialId();
  }, [ trialId, stopPollingFirstTrialId, wontHaveTrials ]);

  useEffect(() => {
    const isPaused = experiment.state === RunState.Paused;
    setHasLoaded(!!trialDetails || isPaused || terminalRunStates.has(experiment.state));
  }, [ experiment.state, trialDetails ]);

  useEffect(() => {
    return () => {
      canceler.abort();
      stopPolling();
      stopPollingFirstTrialId();
    };
  }, [ canceler, stopPolling, stopPollingFirstTrialId ]);

  /*
   * Immediately attempt to fetch trial details instead of waiting for the
   * next polling cycle when trial Id goes from undefined to defined.
   */
  useEffect(() => {
    if (prevTrialId === undefined && prevTrialId !== trialId) fetchTrialDetails();
  }, [ fetchTrialDetails, prevTrialId, trialId ]);

  if (!hasLoaded) return <Spinner tip={ trialId === undefined ?
    'Waiting for trial...' : `Fetching trial ${trialId} details...`
  } />;

  return (
    <>
      <Tabs defaultActiveKey={tabKey} onChange={handleTabChange}>
        <TabPane key="overview" tab="Overview">
          {trialDetails
            ? <TrialDetailsOverview experiment={experiment} trial={trialDetails} />
            : NoDataAlert}
        </TabPane>
        <TabPane key="hyperparameters" tab="Hyperparameters">
          {trialDetails
            ? <TrialDetailsHyperparameters experiment={experiment} trial={trialDetails} />
            : NoDataAlert}
        </TabPane>
        <TabPane key="workloads" tab="Workloads">
          {trialDetails
            ? <TrialDetailsWorkloads experiment={experiment} trial={trialDetails} />
            : NoDataAlert}
        </TabPane>
        <TabPane key="configuration" tab="Configuration">
          <React.Suspense fallback={<Spinner tip="Loading text editor..." />}>
            <ExperimentConfiguration experiment={experiment} />
          </React.Suspense>
        </TabPane>
        <TabPane key="profiler" tab="Profiler">
          {trialDetails
            ? <TrialDetailsProfiles experiment={experiment} trial={trialDetails} />
            : NoDataAlert}
        </TabPane>
        <TabPane key="logs" tab="Logs">
          {trialDetails
            ? <TrialDetailsLogs experiment={experiment} trial={trialDetails} />
            : NoDataAlert}
        </TabPane>
      </Tabs>
    </>
  );
};

export default ExperimentSingleTrialTabs;
