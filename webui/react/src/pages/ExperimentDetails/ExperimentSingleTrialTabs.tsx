import { Alert, Tabs } from 'antd';
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
  trialId?: number;
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
    history.replace(`${basePath}/${key}`);
  }, [ basePath, history ]);

  // Sets the default sub route.
  useEffect(() => {
    if (!tab || (tab && !TAB_KEYS.includes(tab))) {
      history.replace(`${basePath}/${tabKey}`);
    }
  }, [ basePath, history, tab, tabKey ]);

  const fetchTrialDetails = useCallback(async () => {
    if (!trialId) return;
    const response = await getTrialDetails({ id: trialId }, { signal: canceler.signal });
    setTrialDetails(response);
  }, [ canceler, trialId ]);

  const { stopPolling } = usePolling(fetchTrialDetails);

  useEffect(() => {
    if (trialDetails && terminalRunStates.has(trialDetails.state)) {
      stopPolling();
    }
  }, [ trialDetails, stopPolling ]);

  return (
    <Tabs defaultActiveKey={tabKey} onChange={handleTabChange}>
      <TabPane key="overview" tab="Overview">
        {trialDetails
          ? <TrialDetailsOverview experiment={experiment} trial={trialDetails} />
          : <Alert message="No data available." type="warning" />
        }
      </TabPane>
      <TabPane key="hyperparameters" tab="Hyperparameters">
        {trialDetails ?
          <TrialDetailsHyperparameters experiment={experiment} trial={trialDetails} />
          : <Alert message="No data available." type="warning" />
        }
      </TabPane>
      <TabPane key="workloads" tab="Workloads">
        {trialDetails ?
          <TrialDetailsWorkloads experiment={experiment} trial={trialDetails} />
          : <Alert message="No data available." type="warning" />
        }
      </TabPane>
      <TabPane key="configuration" tab="Configuration">
        <React.Suspense fallback={<Spinner />}>
          <ExperimentConfiguration experiment={experiment} />
        </React.Suspense>
      </TabPane>
      <TabPane key="profiler" tab="Profiler">
        {trialDetails ?
          <TrialDetailsProfiles experiment={experiment} trial={trialDetails} />
          : <Alert message="No data available." type="warning" />
        }
      </TabPane>
      <TabPane key="logs" tab="Logs">
        {trialDetails ?
          <TrialDetailsLogs experiment={experiment} trial={trialDetails} />
          : <Alert message="No data available." type="warning" />
        }
      </TabPane>
    </Tabs>
  );
};

export default ExperimentSingleTrialTabs;
