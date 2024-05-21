import Message from 'hew/Message';
import Pivot, { PivotProps } from 'hew/Pivot';
import Spinner from 'hew/Spinner';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Page from 'components/Page';
import RemainingRetentionDaysLabel from 'components/RemainingRetentionDaysLabelComponent';
import RoutePagination from 'components/RoutePagination';
import TrialLogPreview from 'components/TrialLogPreview';
import { terminalRunStates } from 'constants/states';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import TrialDetailsHeader from 'pages/TrialDetails/TrialDetailsHeader';
import TrialDetailsHyperparameters from 'pages/TrialDetails/TrialDetailsHyperparameters';
import TrialDetailsLogs from 'pages/TrialDetails/TrialDetailsLogs';
import TrialDetailsMetrics from 'pages/TrialDetails/TrialDetailsMetrics';
import TrialDetailsOverview from 'pages/TrialDetails/TrialDetailsOverview';
import TrialDetailsProfiles from 'pages/TrialDetails/TrialDetailsProfiles';
import { paths } from 'routes/utils';
import {
  getExperimentDetails,
  getTrialDetails,
  getTrialRemainingLogRetentionDays,
} from 'services/api';
import workspaceStore from 'stores/workspaces';
import { ApiState, ExperimentBase, TrialDetails, ValueOf, Workspace } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { isSingleTrialExperiment } from 'utils/experiment';
import { useObservable } from 'utils/observable';
import { isAborted, isNotFound } from 'utils/service';

import MultiTrialDetailsHyperparameters from './TrialDetails/MultiTrialDetailsHyperparameters';

const TabType = {
  Hyperparameters: 'hyperparameters',
  Logs: 'logs',
  Metrics: 'metrics',
  Overview: 'overview',
  Profiler: 'profiler',
  Workloads: 'workloads',
} as const;

type TabType = ValueOf<typeof TabType>;

type Params = {
  experimentId?: string;
  tab?: TabType;
  trialId: string;
};

const TAB_KEYS = Object.values(TabType);
const DEFAULT_TAB_KEY = TabType.Overview;

const TrialDetailsComp: React.FC = () => {
  const [canceler] = useState(new AbortController());
  const [experiment, setExperiment] = useState<ExperimentBase>();
  const [isFetching, setIsFetching] = useState(false);
  const navigate = useNavigate();
  const { experimentId, tab, trialId: trialID } = useParams<Params>();
  const defaultTabKey = tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY;
  const [tabKey, setTabKey] = useState<TabType>(defaultTabKey);
  const [trialId, setTrialId] = useState<number>(Number(trialID));
  const [trialDetails, setTrialDetails] = useState<ApiState<TrialDetails>>({
    data: undefined,
    error: undefined,
  });
  const pageRef = useRef<HTMLElement>(null);
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));
  const basePath = paths.trialDetails(trialId, experimentId);
  const trial = trialDetails.data;
  const [remainingLogDays, setRemainingLogDays] = useState<Loadable<number | undefined>>(NotLoaded);

  const showExperimentArtifacts = usePermissions().canViewExperimentArtifacts({
    workspace: { id: experiment?.workspaceId ?? 0 },
  });

  const fetchExperimentDetails = useCallback(async () => {
    if (!trial) return;

    try {
      const response = await getExperimentDetails(
        { id: trial.experimentId },
        { signal: canceler.signal },
      );

      setExperiment(response);

      // Experiment id does not exist in route, reroute to the one with it
      if (!experimentId) {
        navigate(paths.trialDetails(trial.id, trial.experimentId), { replace: true });
      }
    } catch (e) {
      handleError(e, {
        publicMessage: 'Failed to load experiment details.',
        publicSubject: 'Unable to fetch Trial Experiment Detail',
        silent: false,
        type: ErrorType.Api,
      });
    } finally {
      setIsFetching(false);
    }
  }, [canceler, navigate, experimentId, trial]);

  const fetchTrialData = useCallback(async () => {
    try {
      const [trialDetailResponse, logRemainingResponse] = await Promise.all([
        getTrialDetails({ id: trialId }, { signal: canceler.signal }),
        getTrialRemainingLogRetentionDays({ id: trialId }),
      ]);
      setTrialDetails((prev) => ({ ...prev, data: trialDetailResponse }));
      setRemainingLogDays(Loaded(logRemainingResponse.remainingLogRetentionDays));
    } catch (e) {
      if (!trialDetails.error && !isAborted(e)) {
        setTrialDetails((prev) => ({ ...prev, error: e as Error }));
      }
    }
  }, [canceler, trialDetails.error, trialId]);

  const handleTabChange = useCallback(
    (key: string) => {
      setTabKey(key as TabType);
      navigate(`${basePath}/${key}`, { replace: true });
    },
    [basePath, navigate],
  );

  // Sets the default sub route.
  useEffect(() => {
    if (!tab || (tab && !TAB_KEYS.includes(tab))) {
      if (window.location.pathname.includes(basePath))
        navigate(`${basePath}/${tabKey}`, { replace: true });
    }
  }, [basePath, navigate, tab, tabKey]);

  const handleViewLogs = useCallback(() => {
    setTabKey(TabType.Logs);
    navigate(`${basePath}/${TabType.Logs}?tail`, { replace: true });
  }, [basePath, navigate]);

  const tabItems: PivotProps['items'] = useMemo(() => {
    if (!experiment || !trial) {
      return [];
    }

    const tabs: PivotProps['items'] = [
      {
        children: <TrialDetailsOverview experiment={experiment} trial={trial} />,
        key: TabType.Overview,
        label: 'Overview',
      },
      {
        children: isSingleTrialExperiment(experiment) ? (
          <TrialDetailsHyperparameters pageRef={pageRef} trial={trial} />
        ) : (
          <MultiTrialDetailsHyperparameters
            experiment={experiment}
            pageRef={pageRef}
            trial={trial}
          />
        ),
        key: TabType.Hyperparameters,
        label: 'Hyperparameters',
      },
      {
        children: <TrialDetailsProfiles trial={trial} />,
        key: TabType.Profiler,
        label: 'Profiler',
      },
      {
        children: <TrialDetailsLogs experiment={experiment} trial={trial} />,
        key: TabType.Logs,
        label: (
          <RemainingRetentionDaysLabel
            remainingLogDays={Loadable.getOrElse(undefined, remainingLogDays)}
          />
        ),
      },
    ];

    if (showExperimentArtifacts) {
      tabs.splice(1, 0, {
        children: <TrialDetailsMetrics experiment={experiment} trial={trial} />,
        key: TabType.Metrics,
        label: 'Metrics',
      });
    }

    return tabs;
  }, [experiment, trial, remainingLogDays, showExperimentArtifacts]);

  const { stopPolling } = usePolling(fetchTrialData);

  useEffect(() => {
    setTrialId(Number(trialID));
  }, [trialID]);

  useEffect(() => {
    setIsFetching(true);
    fetchTrialData();
  }, [fetchTrialData, trialId]);

  useEffect(() => {
    fetchExperimentDetails();
  }, [fetchExperimentDetails]);

  useEffect(() => {
    if (trialDetails.data && terminalRunStates.has(trialDetails.data.state)) {
      stopPolling();
    }
  }, [trialDetails.data, stopPolling]);

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  if (isNaN(trialId)) {
    return <Message title={`Invalid Trial ID ${trialId}`} />;
  }

  if (trialDetails.error !== undefined && !isNotFound(trialDetails.error)) {
    const message = `Unable to fetch Trial ${trialId}`;
    return <Message description={trialDetails.error.message} icon="warning" title={message} />;
  }

  if (!trial || !experiment) {
    return <Spinner spinning tip={`Fetching ${trial ? 'experiment' : 'trial'} information...`} />;
  }

  const workspaceName = workspaces.find((ws: Workspace) => ws.id === experiment?.workspaceId)?.name;

  return (
    <Page
      breadcrumb={[
        workspaceName && experiment?.workspaceId !== 1
          ? {
              breadcrumbName: workspaceName,
              path: paths.workspaceDetails(experiment?.workspaceId ?? 1),
            }
          : {
              breadcrumbName: 'Uncategorized Experiments',
              path: paths.projectDetails(1),
            },
        {
          breadcrumbName: experiment?.name ?? '',
          path: paths.experimentDetails(experiment.id),
        },
        {
          breadcrumbName: `Trial ${trial.id}`,
          path: paths.trialDetails(trial.id),
        },
      ]}
      containerRef={pageRef}
      headerComponent={
        <TrialDetailsHeader
          experiment={experiment}
          fetchTrialDetails={fetchTrialData}
          trial={trial}
        />
      }
      notFound={trialDetails.error && isNotFound(trialDetails.error)}
      stickyHeader
      title={`Trial ${trialId}`}>
      <TrialLogPreview
        hidePreview={tabKey === TabType.Logs}
        trial={trial}
        onViewLogs={handleViewLogs}>
        <Spinner spinning={isFetching}>
          <Pivot
            activeKey={tabKey}
            destroyInactiveTabPane
            items={tabItems}
            tabBarExtraContent={
              <RoutePagination
                currentId={trialId}
                ids={experiment.trialIds ?? []}
                tooltipLabel="Trial"
                onSelectId={(selectedTrialId) => {
                  navigate(paths.trialDetails(selectedTrialId, experiment?.id));
                }}
              />
            }
            onChange={handleTabChange}
          />
        </Spinner>
      </TrialLogPreview>
    </Page>
  );
};

export default TrialDetailsComp;
