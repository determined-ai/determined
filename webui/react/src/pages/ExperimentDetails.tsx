import { Space, Tabs } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useHistory, useLocation, useParams } from 'react-router';

import Badge, { BadgeType } from 'components/Badge';
import CreateExperimentModal, { CreateExperimentType } from 'components/CreateExperimentModal';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import usePolling from 'hooks/usePolling';
import ExperimentActions from 'pages/ExperimentDetails/ExperimentActions';
import ExperimentDetailsHeader from 'pages/ExperimentDetails/ExperimentDetailsHeader';
import ExperimentOverview from 'pages/ExperimentDetails/ExperimentOverview';
import { paths, routeAll } from 'routes/utils';
import { getExperimentDetails, getExpValidationHistory, isNotFound } from 'services/api';
import { createExperiment } from 'services/api';
import { isAborted } from 'services/utils';
import { ExperimentBase, ExperimentVisualizationType, RawJson, ValidationHistory } from 'types';
import { clone, isEqual } from 'utils/data';
import { terminalRunStates, upgradeConfig } from 'utils/types';

const { TabPane } = Tabs;

enum TabType {
  Overview = 'overview',
  Visualization = 'visualization',
}

interface Params {
  experimentId: string;
  tab?: TabType;
  viz?: ExperimentVisualizationType;
}

const TAB_KEYS = Object.values(TabType);
const DEFAULT_TAB_KEY = TabType.Overview;

const ExperimentVisualization = React.lazy(() => {
  return import('./ExperimentDetails/ExperimentVisualization');
});

const ExperimentDetails: React.FC = () => {
  const { experimentId, tab, viz } = useParams<Params>();
  const location = useLocation();
  const history = useHistory();
  const defaultTabKey = tab && TAB_KEYS.indexOf(tab) ? tab : DEFAULT_TAB_KEY;
  const [ tabKey, setTabKey ] = useState(defaultTabKey);
  const [ canceler ] = useState(new AbortController());
  const [ experiment, setExperiment ] = useState<ExperimentBase>();
  const [ valHistory, setValHistory ] = useState<ValidationHistory[]>([]);
  const [ pageError, setPageError ] = useState<Error>();
  const [ forkModalConfig, setForkModalConfig ] = useState<RawJson>();
  const [ forkModalError, setForkModalError ] = useState<string>();
  const [ isForkModalVisible, setIsForkModalVisible ] = useState(false);

  const isShowNewHeader: boolean = useMemo(() => {
    const search = new URLSearchParams(location.search);
    return search.get('header') === 'new';
  }, [ location.search ]);

  const id = parseInt(experimentId);
  const basePath = paths.experimentDetails(experimentId);

  const fetchExperimentDetails = useCallback(async () => {
    try {
      const [ experimentData, validationHistory ] = await Promise.all([
        getExperimentDetails({ id }, { signal: canceler.signal }),
        getExpValidationHistory({ id }, { signal: canceler.signal }),
      ]);
      if (!isEqual(experimentData, experiment)) setExperiment(experimentData);
      if (!isEqual(validationHistory, valHistory)) setValHistory(validationHistory);
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e);
    }
  }, [
    experiment,
    id,
    canceler.signal,
    pageError,
    valHistory,
  ]);

  const { startPolling, stopPolling } = usePolling(fetchExperimentDetails);

  const handleTabChange = useCallback(key => {
    setTabKey(key);
    history.replace(key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`);
  }, [ basePath, history ]);

  const showForkModal = useCallback((): void => {
    if (experiment?.configRaw) {
      const rawConfig: RawJson = clone(experiment.configRaw);
      if (rawConfig.description) rawConfig.description = `Fork of ${rawConfig.description}`;
      upgradeConfig(rawConfig);
      setForkModalConfig(rawConfig);
    }
    setIsForkModalVisible(true);
  }, [ experiment?.configRaw ]);

  const handleForkModalCancel = useCallback(() => {
    setIsForkModalVisible(false);
  }, []);

  const handleForkModalSubmit = useCallback(async (newConfig: string) => {
    try {
      const { id: configId } = await createExperiment({
        experimentConfig: newConfig,
        parentId: id,
      });

      // Reset experiment state and start polling for newly forked experiment.
      setIsForkModalVisible(false);
      setExperiment(undefined);

      // Route to newly forkex experiment.
      routeAll(paths.experimentDetails(configId));

      // Add a slight delay to allow polling function to update.
      setTimeout(() => startPolling(), 100);
    } catch (e) {
      setForkModalError(e.response?.data?.message || 'Unable to create experiment.');
      let errorMessage = 'Unable to fork experiment with the provided config.';
      if (e.name === 'YAMLException') {
        errorMessage = e.message;
      } else if (e.response?.data?.message) {
        errorMessage = e.response.data.message;
      }
      setForkModalError(errorMessage);
    }
  }, [ id, startPolling ]);

  useEffect(() => {
    if (tab && (!TAB_KEYS.includes(tab) || tab === DEFAULT_TAB_KEY)) {
      history.replace(basePath);
    }
  }, [ basePath, history, tab ]);

  useEffect(() => {
    if (experiment && terminalRunStates.has(experiment.state)) {
      stopPolling();
    }
  }, [ experiment, stopPolling ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  if (isNaN(id)) {
    return <Message title={`Invalid Experiment ID ${experimentId}`} />;
  } else if (pageError) {
    const message = isNotFound(pageError) ?
      `Unable to find Experiment ${experimentId}` :
      `Unable to fetch Experiment ${experimentId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!experiment) {
    return <Spinner />;
  }

  return (
    <Page
      headerComponent={isShowNewHeader && <ExperimentDetailsHeader
        experiment={experiment}
        fetchExperimentDetails={fetchExperimentDetails}
        showForkModal={showForkModal}
      />}
      options={<ExperimentActions
        experiment={experiment}
        onClick={{ Fork: showForkModal }}
        onSettled={fetchExperimentDetails} />}
      stickyHeader
      subTitle={<Space align="center" size="small">
        {experiment?.config.name}
        <Badge state={experiment.state} type={BadgeType.State} />
        {experiment.archived && <Badge>ARCHIVED</Badge>}
      </Space>}
      title={`Experiment ${experimentId}`}>
      <Tabs defaultActiveKey={tabKey} onChange={handleTabChange}>
        <TabPane key="overview" tab="Overview">
          <ExperimentOverview
            experiment={experiment}
            validationHistory={valHistory}
            onTagsChange={fetchExperimentDetails} />
        </TabPane>
        <TabPane key="visualization" tab="Visualization">
          <React.Suspense fallback={<Spinner />}>
            <ExperimentVisualization
              basePath={`${basePath}/${TabType.Visualization}`}
              experiment={experiment}
              type={viz} />
          </React.Suspense>
        </TabPane>
      </Tabs>
      <CreateExperimentModal
        config={forkModalConfig}
        error={forkModalError}
        title={`Fork Experiment ${id}`}
        type={CreateExperimentType.Fork}
        visible={isForkModalVisible}
        onCancel={handleForkModalCancel}
        onOk={handleForkModalSubmit}
      />
    </Page>
  );
};

export default ExperimentDetails;
