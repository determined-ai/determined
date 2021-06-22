import { Tabs } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import Spinner from 'components/Spinner';
import usePolling from 'hooks/usePolling';
import { paths } from 'routes/utils';
import { getTrialDetails } from 'services/api';
import { ExperimentBase, TrialDetails } from 'types';
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
  trialId: number;
}

const ExperimentSingleTrialTabs: React.FC<Props> = (
  { experiment, trialId }: Props,
) => {
  const [ canceler ] = useState(new AbortController());
  const [ trialDetails, setTrialDetails ] = useState<TrialDetails>();
  const history = useHistory();
  const { tab } = useParams<Params>();

  const basePath = paths.experimentDetails(experiment.id);
  const defaultTabKey = tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY;

  const [ tabKey, setTabKey ] = useState(defaultTabKey);

  const handleTabChange = useCallback(key => {
    setTabKey(key);
    history.replace(key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`);
  }, [ basePath, history ]);

  useEffect(() => {
    if (tab && (!TAB_KEYS.includes(tab) || tab === DEFAULT_TAB_KEY)) {
      history.replace(basePath);
    }
  }, [ basePath, history, tab ]);

  const fetchTrialDetails = useCallback(async () => {
    const response = await getTrialDetails({ id: trialId }, { signal: canceler.signal });
    setTrialDetails(response);
  }, [ canceler, trialId ]);

  const { stopPolling } = usePolling(fetchTrialDetails);

  useEffect(() => {
    if (trialDetails && terminalRunStates.has(trialDetails.state)) {
      stopPolling();
    }
  }, [ trialDetails, stopPolling ]);

  if (!trialDetails) {
    return <Spinner />;
  }

  return (
    <Tabs defaultActiveKey={tabKey} onChange={handleTabChange}>
      <TabPane key="overview" tab="Overview">
        <TrialDetailsOverview experiment={experiment} trial={trialDetails} />
      </TabPane>
      <TabPane key="hyperparameters" tab="Hyperparameters">
        <TrialDetailsHyperparameters experiment={experiment} trial={trialDetails} />
      </TabPane>
      <TabPane key="workloads" tab="Workloads">
        <TrialDetailsWorkloads experiment={experiment} trial={trialDetails} />
      </TabPane>
      <TabPane key="configuration" tab="Configuration">
        <React.Suspense fallback={<Spinner />}>
          <ExperimentConfiguration experiment={experiment} />
        </React.Suspense>
      </TabPane>
      <TabPane key="profiler" tab="Profiler">
        <TrialDetailsProfiles experiment={experiment} trial={trialDetails} />
      </TabPane>
      <TabPane key="logs" tab="Logs">
        <TrialDetailsLogs experiment={experiment} trial={trialDetails} />
      </TabPane>
    </Tabs>
  );
};

export default ExperimentSingleTrialTabs;
