import { Tabs } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Page from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import RoutePagination from 'components/RoutePagination';
import TrialLogPreview from 'components/TrialLogPreview';
import { terminalRunStates } from 'constants/states';
import TrialDetailsHeader from 'pages/TrialDetails/TrialDetailsHeader';
import TrialDetailsHyperparameters from 'pages/TrialDetails/TrialDetailsHyperparameters';
import TrialDetailsLogs from 'pages/TrialDetails/TrialDetailsLogs';
import TrialDetailsOverview from 'pages/TrialDetails/TrialDetailsOverview';
import TrialDetailsProfiles from 'pages/TrialDetails/TrialDetailsProfiles';
import TrialRangeHyperparameters from 'pages/TrialDetails/TrialRangeHyperparameters';
import { paths } from 'routes/utils';
import { getExperimentDetails, getTrialDetails } from 'services/api';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import usePolling from 'shared/hooks/usePolling';
import { ApiState, ValueOf } from 'shared/types';
import { ErrorType } from 'shared/utils/error';
import { isAborted, isNotFound } from 'shared/utils/service';
import { ExperimentBase, TrialDetails } from 'types';
import handleError from 'utils/error';
import { isSingleTrialExperiment } from 'utils/experiment';

const { TabPane } = Tabs;

const TabType = {
  Hyperparameters: 'hyperparameters',
  Logs: 'logs',
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

  const basePath = paths.trialDetails(trialId, experimentId);
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

  if (trialDetails.error !== undefined) {
    if (isNotFound(trialDetails.error)) return <PageNotFound />;
    const message = `Unable to fetch Trial ${trialId}`;
    return (
      <Message message={trialDetails.error.message} title={message} type={MessageType.Warning} />
    );
  }

  if (!trial || !experiment) {
    return <Spinner tip={`Fetching ${trial ? 'experiment' : 'trial'} information...`} />;
  }

  return (
    <Page
      bodyNoPadding
      containerRef={pageRef}
      headerComponent={
        <TrialDetailsHeader
          experiment={experiment}
          fetchTrialDetails={fetchTrialDetails}
          trial={trial}
        />
      }
      stickyHeader
      title={`Trial ${trialId}`}>
      <TrialLogPreview
        hidePreview={tabKey === TabType.Logs}
        trial={trial}
        onViewLogs={handleViewLogs}>
        <Spinner spinning={isFetching}>
          <Tabs
            activeKey={tabKey}
            className="no-padding"
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
            onChange={handleTabChange}>
            <TabPane key={TabType.Overview} tab="Overview">
              <TrialDetailsOverview experiment={experiment} trial={trial} />
            </TabPane>
            <TabPane key={TabType.Hyperparameters} tab="Hyperparameters">
              {isSingleTrialExperiment(experiment) ? (
                <TrialDetailsHyperparameters pageRef={pageRef} trial={trial} />
              ) : (
                <TrialRangeHyperparameters experiment={experiment} trial={trial} />
              )}
            </TabPane>
            <TabPane key={TabType.Profiler} tab="Profiler">
              <TrialDetailsProfiles experiment={experiment} trial={trial} />
            </TabPane>
            <TabPane key={TabType.Logs} tab="Logs">
              <TrialDetailsLogs experiment={experiment} trial={trial} />
            </TabPane>
          </Tabs>
        </Spinner>
      </TrialLogPreview>
    </Page>
  );
};

export default TrialDetailsComp;
