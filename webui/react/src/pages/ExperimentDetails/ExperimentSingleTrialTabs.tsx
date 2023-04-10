import type { TabsProps } from 'antd';
import { string } from 'io-ts';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';

import Button from 'components/kit/Button';
import Pivot from 'components/kit/Pivot';
import NotesCard from 'components/NotesCard';
import TrialLogPreview from 'components/TrialLogPreview';
import { terminalRunStates } from 'constants/states';
import useFeature from 'hooks/useFeature';
import useModalHyperparameterSearch from 'hooks/useModal/HyperparameterSearch/useModalHyperparameterSearch';
import usePermissions from 'hooks/usePermissions';
import { SettingsConfig, useSettings } from 'hooks/useSettings';
import F_TrialDetailsOverview from 'pages/TrialDetails/F_TrialDetailsOverview';
import { paths } from 'routes/utils';
import { getExpTrials, getTrialDetails, patchExperiment } from 'services/api';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner/Spinner';
import usePolling from 'shared/hooks/usePolling';
import usePrevious from 'shared/hooks/usePrevious';
import { ValueOf } from 'shared/types';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { ExperimentBase, TrialDetails, TrialItem } from 'types';
import handleError from 'utils/error';

import TrialDetailsHyperparameters from '../TrialDetails/TrialDetailsHyperparameters';
import TrialDetailsLogs from '../TrialDetails/TrialDetailsLogs';
import TrialDetailsOverview from '../TrialDetails/TrialDetailsOverview';
import TrialDetailsProfiles from '../TrialDetails/TrialDetailsProfiles';

import ExperimentCheckpoints from './ExperimentCheckpoints';

const CodeEditor = React.lazy(() => import('components/kit/CodeEditor'));

const TabType = {
  Checkpoints: 'checkpoints',
  Code: 'code',
  Hyperparameters: 'hyperparameters',
  Logs: 'logs',
  Notes: 'notes',
  Overview: 'overview',
  Profiler: 'profiler',
  Workloads: 'workloads',
} as const;

type Params = {
  tab?: ValueOf<typeof TabType>;
};

const NeverTrials: React.FC = () => (
  <Message title="Experiment will not have trials" type={MessageType.Alert} />
);

const TAB_KEYS = Object.values(TabType);
const DEFAULT_TAB_KEY = TabType.Overview;

export interface Props {
  experiment: ExperimentBase;
  fetchExperimentDetails: () => void;
  onTrialUpdate?: (trial: TrialItem) => void;
  pageRef: React.RefObject<HTMLElement>;
}

const ExperimentSingleTrialTabs: React.FC<Props> = ({
  experiment,
  fetchExperimentDetails,
  onTrialUpdate,
  pageRef,
}: Props) => {
  const navigate = useNavigate();
  const location = useLocation();
  const [trialId, setFirstTrialId] = useState<number>();
  const [wontHaveTrials, setWontHaveTrials] = useState<boolean>(false);
  const prevTrialId = usePrevious(trialId, undefined);
  const { tab } = useParams<Params>();
  const [canceler] = useState(new AbortController());
  const [trialDetails, setTrialDetails] = useState<TrialDetails>();
  const [tabKey, setTabKey] = useState(tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY);
  const {
    contextHolder: modalHyperparameterSearchContextHolder,
    modalOpen: openHyperparameterSearchModal,
  } = useModalHyperparameterSearch({ experiment });
  const chartFlagOn = useFeature().isOn('chart');

  const waitingForTrials = !trialId && !wontHaveTrials;

  const basePath = paths.experimentDetails(experiment.id);

  const configForExperiment = (experimentId: number): SettingsConfig<{ filePath: string }> => ({
    settings: {
      filePath: {
        defaultValue: '',
        storageKey: 'filePath',
        type: string,
      },
    },
    storagePath: `selected-file-${experimentId}`,
  });
  const { settings, updateSettings } = useSettings<{ filePath: string }>(
    configForExperiment(experiment.id),
  );
  const handleSelectFile = useCallback(
    (filePath: string) => {
      updateSettings({ filePath });
    },
    [updateSettings],
  );

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
  }, [canceler, experiment.id, experiment.state, onTrialUpdate]);

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
  }, [canceler.signal, onTrialUpdate, trialId]);

  const { stopPolling } = usePolling(fetchTrialDetails, { rerunOnNewFn: true });
  const { stopPolling: stopPollingFirstTrialId } = usePolling(fetchFirstTrialId, {
    rerunOnNewFn: true,
  });

  const handleTabChange = useCallback(
    (key: string) => {
      navigate(`${basePath}/${key}`, { replace: true });
    },
    [basePath, navigate],
  );

  const handleViewLogs = useCallback(() => {
    setTabKey(TabType.Logs);
    navigate(`${basePath}/${TabType.Logs}?tail`, { replace: true });
  }, [basePath, navigate]);

  useEffect(() => {
    setTabKey(tab ?? DEFAULT_TAB_KEY);
  }, [location.pathname, tab]);

  // Sets the default sub route.
  useEffect(() => {
    if (!tab || (tab && !TAB_KEYS.includes(tab))) {
      if (window.location.pathname.includes(basePath))
        navigate(`${basePath}/${tabKey}`, { replace: true });
    }
  }, [basePath, navigate, tab, tabKey]);

  useEffect(() => {
    if (trialDetails && terminalRunStates.has(trialDetails.state)) {
      stopPolling();
    }
  }, [trialDetails, stopPolling]);

  useEffect(() => {
    if (wontHaveTrials || trialId !== undefined) stopPollingFirstTrialId();
  }, [trialId, stopPollingFirstTrialId, wontHaveTrials]);

  useEffect(() => {
    return () => {
      canceler.abort();
      stopPolling();
      stopPollingFirstTrialId();
    };
  }, [canceler, stopPolling, stopPollingFirstTrialId]);

  /*
   * Immediately attempt to fetch trial details instead of waiting for the
   * next polling cycle when trial Id goes from undefined to defined.
   */
  useEffect(() => {
    if (prevTrialId === undefined && prevTrialId !== trialId) fetchTrialDetails();
  }, [fetchTrialDetails, prevTrialId, trialId]);

  const handleNotesUpdate = useCallback(
    async (editedNotes: string) => {
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
    },
    [experiment.id, fetchExperimentDetails],
  );

  const handleHPSearch = useCallback(() => {
    openHyperparameterSearchModal({});
  }, [openHyperparameterSearchModal]);

  const { canCreateExperiment, canModifyExperimentMetadata, canViewExperimentArtifacts } =
    usePermissions();
  const workspace = { id: experiment.workspaceId };
  const editableNotes = canModifyExperimentMetadata({ workspace });
  const showExperimentArtifacts = canViewExperimentArtifacts({ workspace });
  const showCreateExperiment = canCreateExperiment({ workspace }) && showExperimentArtifacts;

  const tabItems: TabsProps['items'] = useMemo(() => {
    const items: TabsProps['items'] = [
      {
        children: waitingForTrials ? (
          <Spinner spinning={true} tip="Waiting for trials..." />
        ) : wontHaveTrials ? (
          <NeverTrials />
        ) : chartFlagOn ? (
          <F_TrialDetailsOverview experiment={experiment} trial={trialDetails} />
        ) : (
          <TrialDetailsOverview experiment={experiment} trial={trialDetails} />
        ),
        key: TabType.Overview,
        label: 'Overview',
      },
      {
        children: wontHaveTrials ? (
          <NeverTrials />
        ) : (
          <TrialDetailsHyperparameters pageRef={pageRef} trial={trialDetails as TrialDetails} />
        ),
        key: TabType.Hyperparameters,
        label: 'Hyperparameters',
      },
    ];

    if (showExperimentArtifacts) {
      items.push({
        children: <ExperimentCheckpoints experiment={experiment} pageRef={pageRef} />,
        key: TabType.Checkpoints,
        label: 'Checkpoints',
      });
      items.push({
        children: (
          <React.Suspense fallback={<Spinner tip="Loading code viewer..." />}>
            <CodeEditor
              files={[]}
              readonly={true}
              selectedFilePath={settings.filePath}
              onSelectFile={handleSelectFile}
            />
          </React.Suspense>
        ),
        key: TabType.Code,
        label: 'Code',
      });
    }

    items.push({
      children: (
        <NotesCard
          disabled={!editableNotes}
          notes={experiment.notes ?? ''}
          style={{ border: 0 }}
          onSave={handleNotesUpdate}
        />
      ),
      key: TabType.Notes,
      label: 'Notes',
    });

    items.push({
      children: (
        <TrialDetailsProfiles experiment={experiment} trial={trialDetails as TrialDetails} />
      ),
      key: TabType.Profiler,
      label: 'Profiler',
    });

    if (showExperimentArtifacts) {
      items.push({
        children: wontHaveTrials ? (
          <NeverTrials />
        ) : (
          <TrialDetailsLogs experiment={experiment} trial={trialDetails as TrialDetails} />
        ),
        key: TabType.Logs,
        label: 'Logs',
      });
    }

    return items;
  }, [
    editableNotes,
    experiment,
    handleNotesUpdate,
    handleSelectFile,
    pageRef,
    settings.filePath,
    showExperimentArtifacts,
    trialDetails,
    waitingForTrials,
    wontHaveTrials,
    chartFlagOn,
  ]);

  return (
    <TrialLogPreview
      hidePreview={tabKey === TabType.Logs}
      trial={trialDetails}
      onViewLogs={handleViewLogs}>
      <Pivot
        activeKey={tabKey}
        items={tabItems}
        tabBarExtraContent={
          tabKey === TabType.Hyperparameters && showCreateExperiment ? (
            <div style={{ padding: 8 }}>
              <Button onClick={handleHPSearch}>Hyperparameter Search</Button>
            </div>
          ) : undefined
        }
        onChange={handleTabChange}
      />
      {modalHyperparameterSearchContextHolder}
    </TrialLogPreview>
  );
};

export default ExperimentSingleTrialTabs;
