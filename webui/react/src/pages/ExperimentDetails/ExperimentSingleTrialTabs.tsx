import { Button, Tabs } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom-v5-compat';

import NotesCard from 'components/NotesCard';
import TrialLogPreview from 'components/TrialLogPreview';
import { terminalRunStates } from 'constants/states';
import useModalHyperparameterSearch
  from 'hooks/useModal/HyperparameterSearch/useModalHyperparameterSearch';
import usePolling from 'hooks/usePolling';
import { paths } from 'routes/utils';
import { getExpTrials, getTrialDetails, patchExperiment } from 'services/api';
import Spinner from 'shared/components/Spinner/Spinner';
import usePrevious from 'shared/hooks/usePrevious';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { ExperimentBase, TrialDetails, TrialItem } from 'types';
import handleError from 'utils/error';

import TrialDetailsHyperparameters from '../TrialDetails/TrialDetailsHyperparameters';
import TrialDetailsLogs from '../TrialDetails/TrialDetailsLogs';
import TrialDetailsOverview from '../TrialDetails/TrialDetailsOverview';
import TrialDetailsProfiles from '../TrialDetails/TrialDetailsProfiles';

const CodeViewer = React.lazy(() => import('./CodeViewer/CodeViewer'));

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

type Params = {
  tab?: TabType;
}

const TAB_KEYS = Object.values(TabType);
const DEFAULT_TAB_KEY = TabType.Overview;

export interface Props {
  experiment: ExperimentBase;
  fetchExperimentDetails: () => void;
  onTrialUpdate?: (trial: TrialItem) => void;
  pageRef: React.RefObject<HTMLElement>;
}

const ExperimentSingleTrialTabs: React.FC<Props> = (
  { experiment, fetchExperimentDetails, onTrialUpdate, pageRef }: Props,
) => {
  const navigate = useNavigate();
  const [ trialId, setFirstTrialId ] = useState<number>();
  const [ wontHaveTrials, setWontHaveTrials ] = useState<boolean>(false);
  const prevTrialId = usePrevious(trialId, undefined);
  const { tab } = useParams<Params>();
  const [ canceler ] = useState(new AbortController());
  const [ trialDetails, setTrialDetails ] = useState<TrialDetails>();
  const [ tabKey, setTabKey ] = useState(tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY);
  const {
    contextHolder: modalHyperparameterSearchContextHolder,
    modalOpen: openHyperparameterSearchModal,
  } = useModalHyperparameterSearch({ experiment });

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
  }, [ canceler.signal, onTrialUpdate, trialId ]);

  const { stopPolling } = usePolling(fetchTrialDetails, { rerunOnNewFn: true });
  const { stopPolling: stopPollingFirstTrialId } = usePolling(
    fetchFirstTrialId,
    { rerunOnNewFn: true },
  );

  const handleTabChange = useCallback((key) => {
    setTabKey(key);
    navigate(`${basePath}/${key}`, { replace: true });
  }, [ basePath, navigate ]);

  const handleViewLogs = useCallback(() => {
    setTabKey(TabType.Logs);
    navigate(`${basePath}/${TabType.Logs}?tail`, { replace: true });
  }, [ basePath, navigate ]);

  // Sets the default sub route.
  useEffect(() => {
    if (!tab || (tab && !TAB_KEYS.includes(tab))) {
      navigate(`${basePath}/${tabKey}`, { replace: true });
    }
  }, [ basePath, navigate, tab, tabKey ]);

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

  const handleHPSearch = useCallback(() => {
    openHyperparameterSearchModal({});
  }, [ openHyperparameterSearchModal ]);

  return (
    <TrialLogPreview
      hidePreview={tabKey === TabType.Logs}
      trial={trialDetails}
      onViewLogs={handleViewLogs}>
      <Tabs
        activeKey={tabKey}
        tabBarExtraContent={tabKey === 'hyperparameters' ? (
          <div style={{ padding: 8 }}>
            <Button onClick={handleHPSearch}>Hyperparameter Search</Button>
          </div>
        ) :
          undefined}
        tabBarStyle={{ height: 48, paddingLeft: 16 }}
        onChange={handleTabChange}>
        <TabPane key="overview" tab="Overview">
          <TrialDetailsOverview experiment={experiment} trial={trialDetails as TrialDetails} />
        </TabPane>
        <TabPane key="hyperparameters" tab="Hyperparameters">
          <TrialDetailsHyperparameters
            pageRef={pageRef}
            trial={trialDetails as TrialDetails}
          />
        </TabPane>
        <TabPane key="code" tab="Code">
          <React.Suspense fallback={<Spinner tip="Loading code viewer..." />}>
            <CodeViewer
              experimentId={experiment.id}
              runtimeConfig={experiment.configRaw}
              submittedConfig={experiment.originalConfig}
            />
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
      {modalHyperparameterSearchContextHolder}
    </TrialLogPreview>
  );
};

export default ExperimentSingleTrialTabs;
