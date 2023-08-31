import type { TabsProps } from 'antd';
import React, { ReactNode, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Pivot from 'components/kit/Pivot';
import Spinner from 'components/kit/Spinner';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import RoutePagination from 'components/RoutePagination';
import TrialLogPreview from 'components/TrialLogPreview';
import { terminalRunStates } from 'constants/states';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import MultiTrialDetailsHyperparameters from 'pages/TrialDetails/MultiTrialDetailsHyperparameters';
import TrialDetailsHeader from 'pages/TrialDetails/TrialDetailsHeader';
import TrialDetailsHyperparameters from 'pages/TrialDetails/TrialDetailsHyperparameters';
import TrialDetailsLogs from 'pages/TrialDetails/TrialDetailsLogs';
import TrialDetailsMetrics from 'pages/TrialDetails/TrialDetailsMetrics';
import TrialDetailsOverview from 'pages/TrialDetails/TrialDetailsOverview';
import TrialDetailsProfiles from 'pages/TrialDetails/TrialDetailsProfiles';
import { paths } from 'routes/utils';
import { getExperimentDetails, getTrialDetails } from 'services/api';
import workspaceStore from 'stores/workspaces';
import { ApiState, ExperimentBase, TrialDetails, ValueOf, Workspace } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { isSingleTrialExperiment } from 'utils/experiment';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';
import { isAborted, isNotFound } from 'utils/service';

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

  const fetchTrialDetails = useCallback(async () => {
    try {
      const response = await getTrialDetails({ id: trialId }, { signal: canceler.signal });
      setTrialDetails((prev) => ({ ...prev, data: response }));
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

  const tabItems: TabsProps['items'] = useMemo(() => {
    if (!experiment || !trial) {
      return [];
    }

    const tabs: Array<{ children: ReactNode; key: TabType; label: string }> = [
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
        children: <TrialDetailsProfiles experiment={experiment} trial={trial} />,
        key: TabType.Profiler,
        label: 'Profiler',
      },
      {
        children: <TrialDetailsLogs experiment={experiment} trial={trial} />,
        key: TabType.Logs,
        label: 'Logs',
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
  }, [experiment, trial, showExperimentArtifacts]);

  const { stopPolling } = usePolling(fetchTrialDetails);

  useEffect(() => {
    setTrialId(Number(trialID));
  }, [trialID]);

  useEffect(() => {
    setIsFetching(true);
    fetchTrialDetails();
  }, [fetchTrialDetails, trialId]);

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
    return (
      <Message message={trialDetails.error.message} title={message} type={MessageType.Warning} />
    );
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
          fetchTrialDetails={fetchTrialDetails}
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
