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

import Badge from 'components/Badge';
import HyperparameterSearchModalComponent from 'components/HyperparameterSearchModal';
import TrialLogPreview from 'components/TrialLogPreview';
import { UNMANAGED_MESSAGE } from 'constant';
import { terminalRunStates } from 'constants/states';
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
import { pluralizer } from 'utils/string';

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
  const [remainingLogDays, setRemainingLogDays] = useState<Loadable<number>>(NotLoaded);

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

  const fetchRemainingLogDays = useCallback(async () => {
    if (!trialId) return;
    try {
      const remainingDays = await getTrialRemainingLogRetentionDays({ id: trialId });
      setRemainingLogDays(
        remainingDays.remainingLogRetentionDays !== undefined
          ? Loaded(remainingDays.remainingLogRetentionDays)
          : NotLoaded,
      );
    } catch (e) {
      setRemainingLogDays(NotLoaded);
    }
  }, [trialId]);

  const fetchAllTrialDetails = useCallback(async () => {
    await Promise.allSettled([fetchTrialDetails(), fetchRemainingLogDays()]);
  }, [fetchRemainingLogDays, fetchTrialDetails]);

  const getRemainingLogsLabel = useMemo(() => {
    let toolTipText = '';
    let badgeText = '';
    const days = Loadable.getOrElse(null, remainingLogDays);
    if (days === null) {
      toolTipText = 'Days remaining to retention are not available yet.';
      badgeText = '-';
    } else if (days === -1) {
      toolTipText = 'Logs will be retained forever.';
      badgeText = 'Frvr';
    } else if (days === 0) {
      toolTipText = 'Some logs have begun to be deleted for this trial.';
      badgeText = '0';
    } else {
      toolTipText = `${pluralizer(days, 'day')} left to retain logs`;
      badgeText = `${remainingLogDays}`;
    }
    return (
      <div>
        <>Logs </>
        <Badge tooltip={toolTipText}>{badgeText}</Badge>
      </div>
    );
  }, [remainingLogDays]);

  const { stopPolling } = usePolling(fetchAllTrialDetails, { rerunOnNewFn: true });
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
    if (prevTrialId === undefined && prevTrialId !== trialId) fetchAllTrialDetails();
  }, [fetchAllTrialDetails, prevTrialId, trialId]);

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
          publicSubject: 'Unable to update experiment notes.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    },
    [experiment.id, fetchExperimentDetails],
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
          <Spinner spinning tip="Waiting for trials..." />
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
        label: getRemainingLogsLabel,
      });
    }

    return items;
  }, [
    editableNotes,
    experiment,
    getRemainingLogsLabel,
    handleNotesUpdate,
    handleSelectFile,
    pageRef,
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
