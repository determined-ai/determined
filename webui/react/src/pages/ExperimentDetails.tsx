import { Space, Tabs } from 'antd';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import Badge, { BadgeType } from 'components/Badge';
import CreateExperimentModalX from 'components/CreateExperimentModalX';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import usePolling from 'hooks/usePolling';
import ExperimentActions from 'pages/ExperimentDetails/ExperimentActions';
import { paths, routeAll } from 'routes/utils';
import { getExperimentDetails, getExpValidationHistory, isNotFound } from 'services/api';
import { createExperiment } from 'services/api';
import { isAborted } from 'services/utils';
import { ExperimentBase, ExperimentVisualizationType, ValidationHistory } from 'types';
import { clone, isEqual } from 'utils/data';
import { terminalRunStates, upgradeConfig } from 'utils/types';

import css from './ExperimentDetails.module.scss';
import ExperimentOverview from './ExperimentDetails/ExperimentOverview';
import ExperimentVisualization from './ExperimentDetails/ExperimentVisualization';

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

const ExperimentDetails: React.FC = () => {
  const { experimentId, tab, viz } = useParams<Params>();
  const history = useHistory();
  const defaultTabKey = tab && TAB_KEYS.indexOf(tab) ? tab : DEFAULT_TAB_KEY;
  const [ tabKey, setTabKey ] = useState(defaultTabKey);
  const [ forkModalError, setForkModalError ] = useState<string>();
  const [ isForkModalVisible, setIsForkModalVisible ] = useState(false);
  const [ forkConfig, setForkConfig ] = useState('');
  const [ canceler ] = useState(new AbortController());
  const [ experiment, setExperiment ] = useState<ExperimentBase>();
  const [ valHistory, setValHistory ] = useState<ValidationHistory[]>([]);
  const [ pageError, setPageError ] = useState<Error>();

  const id = parseInt(experimentId);
  const basePath = paths.experimentDetails(experimentId);

  const fetchExperimentDetails = useCallback(async () => {
    try {
      const [ experimentData, validationHistory ] = await Promise.all([
        getExperimentDetails({ id }, { signal: canceler.signal }),
        getExpValidationHistory({ id }, { signal: canceler.signal }),
      ]);
      if (!isEqual(experimentData, experiment)) {
        setExperiment(experimentData);
      }
      if (!isEqual(validationHistory, valHistory)) {
        setValHistory(validationHistory);
      }
    } catch (e) {
      if (!pageError && !isAborted(e)) {
        setPageError(e);
      }
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
      const rawConfig = clone(experiment.configRaw);
      rawConfig.description = `Fork of ${rawConfig.description}`;
      upgradeConfig(rawConfig);
      setForkConfig(yaml.safeDump(rawConfig));
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
      setIsForkModalVisible(false);

      // Reset experiment state and start polling for newly forked experiment.
      setExperiment(undefined);
      startPolling();
      routeAll(paths.experimentDetails(configId));
    } catch (e) {
      setForkModalError(e.response?.data?.message || 'Unable to create experiment.');
      let errorMessage = 'Unable to fork experiment with the provide config.';
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
      breadcrumb={[
        { breadcrumbName: 'Experiments', path: paths.experimentList() },
        {
          breadcrumbName: `Experiment ${experimentId}`,
          path: paths.experimentDetails(experimentId),
        },
      ]}
      options={<ExperimentActions
        experiment={experiment}
        onClick={{ Fork: showForkModal }}
        onSettled={fetchExperimentDetails} />}
      stickHeader
      subTitle={<Space align="center" size="small">
        {experiment?.config.description}
        <Badge state={experiment.state} type={BadgeType.State} />
        {experiment.archived && <Badge>ARCHIVED</Badge>}
      </Space>}
      title={`Experiment ${experimentId}`}>
      <Tabs className={css.base} defaultActiveKey={tabKey} onChange={handleTabChange}>
        <TabPane key="overview" tab="Overview">
          <ExperimentOverview
            experiment={experiment}
            validationHistory={valHistory}
            onTagsChange={fetchExperimentDetails} />
        </TabPane>
        <TabPane key="visualization" tab="Visualization">
          <ExperimentVisualization
            basePath={`${basePath}/${TabType.Visualization}`}
            experiment={experiment}
            type={viz} />
        </TabPane>
      </Tabs>
      <CreateExperimentModalX
        config={forkConfig}
        error={forkModalError}
        okText="Fork"
        title={`Fork Experiment ${id}`}
        visible={isForkModalVisible}
        onCancel={handleForkModalCancel}
        onOk={handleForkModalSubmit}
      />
    </Page>
  );
};

export default ExperimentDetails;
