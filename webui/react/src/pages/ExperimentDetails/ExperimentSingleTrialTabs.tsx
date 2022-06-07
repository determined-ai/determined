import { Tabs } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import NotesCard from 'components/NotesCard';
import TrialLogPreview from 'components/TrialLogPreview';
import { terminalRunStates } from 'constants/states';
import usePolling from 'hooks/usePolling';
import usePrevious from 'hooks/usePrevious';
import { paths } from 'routes/utils';
import { getExpTrials, getTrialDetails, patchExperiment } from 'services/api';
import Spinner from 'shared/components/Spinner/Spinner';
import { ExperimentBase, TrialDetails } from 'types';
import handleError from 'utils/error';

import { ErrorLevel, ErrorType } from '../../shared/utils/error';
import TrialDetailsHyperparameters from '../TrialDetails/TrialDetailsHyperparameters';
import TrialDetailsLogs from '../TrialDetails/TrialDetailsLogs';
import TrialDetailsOverview from '../TrialDetails/TrialDetailsOverview';
import TrialDetailsProfiles from '../TrialDetails/TrialDetailsProfiles';

const { TabPane } = Tabs;

enum TabType {
  Configuration = 'configuration',
  Hyperparameters = 'hyperparameters',
  Logs = 'logs',
  Overview = 'overview',
  Profiler = 'profiler',
  Workloads = 'workloads',
  Notes = 'notes'
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
  fetchExperimentDetails: () => void;
  onTrialUpdate?: (trial: TrialDetails) => void;
}

const ExperimentSingleTrialTabs: React.FC<Props> = (
  { experiment, fetchExperimentDetails, onTrialUpdate }: Props,
) => {
  const history = useHistory();
  const [ trialId, setFirstTrialId ] = useState<number>();
  const [ wontHaveTrials, setWontHaveTrials ] = useState<boolean>(false);
  const prevTrialId = usePrevious(trialId, undefined);
  const { tab } = useParams<Params>();
  const [ canceler ] = useState(new AbortController());
  const [ trialDetails, setTrialDetails ] = useState<TrialDetails>();
  const [ tabKey, setTabKey ] = useState(tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY);

  const basePath = paths.experimentDetails(experiment.id);

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
        onTrialUpdate?.(firstTrial);
        setFirstTrialId(firstTrial.id);
      } else if (isTerminalExp) {
        setWontHaveTrials(true);
      }
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Failed to fetch experiment trials.',
        silent: true,
        type: ErrorType.Server,
      });
    }
  }, [ canceler, experiment.id, experiment.state, onTrialUpdate ]);

  const fetchTrialDetails = useCallback(async () => {
    if (!trialId) return;
    try {
      const response = await getTrialDetails({ id: trialId }, { signal: canceler.signal });
      onTrialUpdate?.(response);
      setTrialDetails(response);
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Failed to fetch experiment trials.',
        silent: true,
        type: ErrorType.Server,
      });
    }
  }, [ canceler, trialId, onTrialUpdate ]);

  const { stopPolling } = usePolling(fetchTrialDetails);
  const { stopPolling: stopPollingFirstTrialId } = usePolling(fetchFirstTrialId);

  const handleTabChange = useCallback(key => {
    setTabKey(key);
    history.replace(`${basePath}/${key}`);
  }, [ basePath, history ]);

  const handleViewLogs = useCallback(() => {
    setTabKey(TabType.Logs);
    history.replace(`${basePath}/${TabType.Logs}?tail`);
  }, [ basePath, history ]);

  // Sets the default sub route.
  useEffect(() => {
    if (!tab || (tab && !TAB_KEYS.includes(tab))) {
      history.replace(`${basePath}/${tabKey}`);
    }
  }, [ basePath, history, tab, tabKey ]);

  useEffect(() => {
    if (trialDetails && terminalRunStates.has(trialDetails.state)) {
      stopPolling();
    }
  }, [ trialDetails, stopPolling ]);

  useEffect(() => {
    if (wontHaveTrials || trialId !== undefined) stopPollingFirstTrialId();
  }, [ trialId, stopPollingFirstTrialId, wontHaveTrials ]);

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

  const handleNotesUpdate = useCallback(async (editedNotes: string) => {
    try {
      await patchExperiment({ body: { notes: editedNotes }, experimentId: experiment.id });
      await fetchExperimentDetails();
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to update experiment notes.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ experiment.id, fetchExperimentDetails ]);

  return (
    <TrialLogPreview
      hidePreview={tabKey === TabType.Logs}
      trial={trialDetails}
      onViewLogs={handleViewLogs}>
      <Tabs activeKey={tabKey} className="no-padding" onChange={handleTabChange}>
        <TabPane key="overview" tab="Overview">
          <TrialDetailsOverview experiment={experiment} trial={trialDetails as TrialDetails} />
        </TabPane>
        <TabPane key="hyperparameters" tab="Hyperparameters">
          <TrialDetailsHyperparameters
            experiment={experiment}
            trial={trialDetails as TrialDetails}
          />
        </TabPane>
        <TabPane key="configuration" tab="Configuration">
          <React.Suspense fallback={<Spinner tip="Loading text editor..." />}>
            <ExperimentConfiguration experiment={experiment} />
          </React.Suspense>
        </TabPane>
        <TabPane key="notes" tab="Notes">
          <NotesCard
            notes={experiment.notes ?? ''}
            style={{ border: 0, height: '100%' }}
            onSave={handleNotesUpdate}
          />
        </TabPane>
        <TabPane key="profiler" tab="Profiler">
          <TrialDetailsProfiles experiment={experiment} trial={trialDetails as TrialDetails} />
        </TabPane>
        <TabPane key="logs" tab="Logs">
          <TrialDetailsLogs experiment={experiment} trial={trialDetails as TrialDetails} />
        </TabPane>
      </Tabs>
    </TrialLogPreview>
  );
};

export default ExperimentSingleTrialTabs;
