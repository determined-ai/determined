import Button from 'hew/Button';
import Message from 'hew/Message';
import { useModal } from 'hew/Modal';
import Pivot, { PivotProps } from 'hew/Pivot';
import Notes from 'hew/RichTextEditor';
import Spinner from 'hew/Spinner';
import Tooltip from 'hew/Tooltip';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { string } from 'io-ts';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { unstable_useBlocker, useLocation, useNavigate, useParams } from 'react-router-dom';

import HyperparameterSearchModalComponent from 'components/HyperparameterSearchModal';
import RemainingRetentionDaysLabel from 'components/RemainingRetentionDaysLabelComponent';
import TrialLogPreview from 'components/TrialLogPreview';
import { UNMANAGED_MESSAGE } from 'constant';
import { terminalRunStates } from 'constants/states';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import usePrevious from 'hooks/usePrevious';
import { SettingsConfig, useSettings } from 'hooks/useSettings';
import TrialDetailsHyperparameters from 'pages/TrialDetails/TrialDetailsHyperparameters';
import TrialDetailsLogs from 'pages/TrialDetails/TrialDetailsLogs';
import TrialDetailsMetrics from 'pages/TrialDetails/TrialDetailsMetrics';
import TrialDetailsOverview from 'pages/TrialDetails/TrialDetailsOverview';
import TrialDetailsProfiles from 'pages/TrialDetails/TrialDetailsProfiles';
import { paths } from 'routes/utils';
import {
  getExpTrials,
  getTrialDetails,
  getTrialRemainingLogRetentionDays,
  patchExperiment,
} from 'services/api';
import { ExperimentBase, Note, TrialDetails, TrialItem, ValueOf } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

import ExperimentCheckpoints from './ExperimentCheckpoints';
import ExperimentCodeViewer from './ExperimentCodeViewer';
import css from './ExperimentSingleTrialTabs.module.scss';

const TabType = {
  Checkpoints: 'checkpoints',
  Code: 'code',
  Hyperparameters: 'hyperparameters',
  Logs: 'logs',
  Metrics: 'metrics',
  Notes: 'notes',
  Overview: 'overview',
  Profiler: 'profiler',
  Workloads: 'workloads',
} as const;

type Params = {
  tab?: ValueOf<typeof TabType>;
};

const NeverTrials: React.FC = () => (
  <Message icon="warning" title="Experiment will not have trials" />
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
  const [remainingLogDays, setRemainingLogDays] = useState<Loadable<number | undefined>>(NotLoaded);
  const f_flat_runs = useFeature().isOn('flat_runs');

  const waitingForTrials = !trialId && !wontHaveTrials;

  const HyperparameterSearchModal = useModal(HyperparameterSearchModalComponent);

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

  const config: SettingsConfig<{ filePath: string }> = useMemo(() => {
    return configForExperiment(experiment.id);
  }, [experiment.id]);
  const { settings, updateSettings } = useSettings<{ filePath: string }>(config);
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
        publicMessage: `Failed to fetch ${f_flat_runs ? 'run' : 'experiment trials'}.`,
        silent: true,
        type: ErrorType.Server,
      });
    }
  }, [canceler, experiment.id, experiment.state, f_flat_runs, onTrialUpdate]);

  const fetchTrialData = useCallback(async () => {
    if (!trialId) return;
    try {
      const [trialDetailResponse, logRemainingResponse] = await Promise.all([
        getTrialDetails({ id: trialId }, { signal: canceler.signal }),
        getTrialRemainingLogRetentionDays({ id: trialId }),
      ]);
      onTrialUpdate?.(trialDetailResponse);
      setTrialDetails(trialDetailResponse);
      setRemainingLogDays(Loaded(logRemainingResponse.remainingLogRetentionDays));
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: `Failed to fetch ${f_flat_runs ? 'run' : 'experiment trials'}.`,
        silent: true,
        type: ErrorType.Server,
      });
    }
  }, [canceler.signal, f_flat_runs, onTrialUpdate, trialId]);

  const { stopPolling } = usePolling(fetchTrialData, { rerunOnNewFn: true });
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
    if (prevTrialId === undefined && prevTrialId !== trialId) {
      fetchTrialData();
    }
  }, [fetchTrialData, prevTrialId, trialId]);

  const handleNotesUpdate = useCallback(
    async (notes: Note) => {
      const editedNotes = notes.contents;
      try {
        await patchExperiment({ body: { notes: editedNotes }, experimentId: experiment.id });
        await fetchExperimentDetails();
      } catch (e) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: `Unable to update ${f_flat_runs ? 'run' : 'experiment'} notes.`,
          silent: false,
          type: ErrorType.Server,
        });
      }
    },
    [experiment.id, f_flat_runs, fetchExperimentDetails],
  );

  const { canCreateExperiment, canModifyExperimentMetadata, canViewExperimentArtifacts } =
    usePermissions();
  const workspace = { id: experiment.workspaceId };
  const editableNotes = canModifyExperimentMetadata({ workspace });
  const showExperimentArtifacts = canViewExperimentArtifacts({ workspace });
  const showCreateExperiment = canCreateExperiment({ workspace }) && showExperimentArtifacts;

  const tabItems: PivotProps['items'] = useMemo(() => {
    const items: PivotProps['items'] = [
      {
        children: waitingForTrials ? (
          <Spinner spinning tip={`Waiting for ${f_flat_runs ? 'run' : 'trial'}s...`} />
        ) : wontHaveTrials ? (
          <NeverTrials />
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
      items.splice(1, 0, {
        children: wontHaveTrials ? (
          <NeverTrials />
        ) : (
          <TrialDetailsMetrics experiment={experiment} trial={trialDetails} />
        ),
        key: TabType.Metrics,
        label: 'Metrics',
      });
      items.push({
        children: <ExperimentCheckpoints experiment={experiment} pageRef={pageRef} />,
        key: TabType.Checkpoints,
        label: 'Checkpoints',
      });
      items.push({
        children: (
          <ExperimentCodeViewer
            experiment={experiment}
            selectedFilePath={settings.filePath}
            onSelectFile={handleSelectFile}
          />
        ),
        disabled: experiment.modelDefinitionSize === 0,
        key: TabType.Code,
        label:
          experiment.modelDefinitionSize !== 0 ? (
            'Code'
          ) : (
            <Tooltip content="Code file non-exist.">Code</Tooltip>
          ),
      });
    }

    items.push({
      children: (
        <Notes
          disabled={!editableNotes}
          disableTitle
          docs={{ contents: experiment.notes ?? '', name: 'Notes' }}
          onError={handleError}
          onPageUnloadHook={unstable_useBlocker}
          onSave={handleNotesUpdate}
        />
      ),
      key: TabType.Notes,
      label: 'Notes',
    });

    items.push({
      children: <TrialDetailsProfiles trial={trialDetails as TrialDetails} />,
      disabled: experiment.unmanaged,
      key: TabType.Profiler,
      label: experiment.unmanaged ? (
        <Tooltip content={UNMANAGED_MESSAGE}>Profiler</Tooltip>
      ) : (
        'Profiler'
      ),
    });

    if (showExperimentArtifacts) {
      items.push({
        children: wontHaveTrials ? (
          <NeverTrials />
        ) : (
          <TrialDetailsLogs experiment={experiment} trial={trialDetails as TrialDetails} />
        ),
        key: TabType.Logs,
        label: (
          <RemainingRetentionDaysLabel
            remainingLogDays={Loadable.getOrElse(undefined, remainingLogDays)}
          />
        ),
      });
    }

    return items;
  }, [
    editableNotes,
    experiment,
    f_flat_runs,
    handleNotesUpdate,
    handleSelectFile,
    pageRef,
    remainingLogDays,
    settings.filePath,
    showExperimentArtifacts,
    trialDetails,
    waitingForTrials,
    wontHaveTrials,
  ]);

  return (
    <TrialLogPreview
      hidePreview={tabKey === TabType.Logs}
      trial={trialDetails}
      onViewLogs={handleViewLogs}>
      <div className={css.pivoter}>
        <Pivot
          activeKey={tabKey}
          items={tabItems}
          tabBarExtraContent={
            tabKey === TabType.Hyperparameters && showCreateExperiment && !experiment.unmanaged ? (
              <div style={{ padding: 4 }}>
                <Button onClick={HyperparameterSearchModal.open}>Hyperparameter Search</Button>
              </div>
            ) : undefined
          }
          onChange={handleTabChange}
        />
      </div>
      <HyperparameterSearchModal.Component
        closeModal={HyperparameterSearchModal.close}
        experiment={experiment}
      />
    </TrialLogPreview>
  );
};

export default ExperimentSingleTrialTabs;
