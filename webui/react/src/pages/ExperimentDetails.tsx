import { Space, Tabs } from 'antd';
import axios from 'axios';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import Badge, { BadgeType } from 'components/Badge';
import CreateExperimentModal from 'components/CreateExperimentModal';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import ExperimentActions from 'pages/ExperimentDetails/ExperimentActions';
import { getExperimentDetails, getExpValidationHistory, isNotFound } from 'services/api';
import { ApiState } from 'services/types';
import { isAborted } from 'services/utils';
import { ExperimentBase, ValidationHistory } from 'types';
import { clone, isEqual } from 'utils/data';
import { terminalRunStates, upgradeConfig } from 'utils/types';

import css from './ExperimentDetails.module.scss';
import ExperimentOverview from './ExperimentDetails/ExperimentOverview';
import ExperimentVisualization, {
  VisualizationType,
} from './ExperimentDetails/ExperimentVisualization';

const { TabPane } = Tabs;

enum TabType {
  Overview = 'overview',
  Visualization = 'visualization',
}

interface Params {
  experimentId: string;
  tab?: TabType;
  viz?: VisualizationType;
}

const TAB_KEYS = Object.values(TabType);
const DEFAULT_TAB_KEY = TabType.Overview;

const ExperimentDetails: React.FC = () => {
  const { experimentId, tab, viz } = useParams<Params>();
  const history = useHistory();
  const defaultTabKey = tab && TAB_KEYS.indexOf(tab) ? tab : DEFAULT_TAB_KEY;
  const [ tabKey, setTabKey ] = useState(defaultTabKey);
  const [ forkModalVisible, setForkModalVisible ] = useState(false);
  const [ forkModalConfig, setForkModalConfig ] = useState('Loading');
  const [ source ] = useState(axios.CancelToken.source());
  const [ experimentDetails, setExperimentDetails ] = useState<ApiState<ExperimentBase>>({
    data: undefined,
    error: undefined,
    isLoading: true,
    source,
  });
  const [ experimentCanceler ] = useState(new AbortController());
  const [ valHistory, setValHistory ] = useState<ValidationHistory[]>([]);

  const id = parseInt(experimentId);
  const basePath = `/experiments/${experimentId}`;
  const experiment = experimentDetails.data;

  const fetchExperimentDetails = useCallback(async () => {
    try {
      const experiment = await getExperimentDetails({ id }, { signal: experimentCanceler.signal });
      const validationHistory = await getExpValidationHistory({ id });
      if (!isEqual(experiment, experimentDetails.data)) {
        setExperimentDetails(prev => ({ ...prev, data: experiment, isLoading: false }));
      }
      setValHistory(validationHistory);
    } catch (e) {
      if (!experimentDetails.error && !isAborted(e)) {
        setExperimentDetails(prev => ({ ...prev, error: e }));
      }
    }
  }, [ id, experimentDetails.data, experimentDetails.error, experimentCanceler.signal ]);

  const setFreshForkConfig = useCallback(() => {
    if (!experiment?.configRaw) return;
    // do not reset the config if the modal is open
    if (forkModalVisible) return;
    const prefix = 'Fork of ';
    const rawConfig = clone(experiment.configRaw);
    rawConfig.description = prefix + rawConfig.description;
    upgradeConfig(rawConfig);
    setForkModalConfig(yaml.safeDump(rawConfig));
  }, [ experiment?.configRaw, forkModalVisible ]);

  const handleForkModalCancel = useCallback(() => {
    setForkModalVisible(false);
    setFreshForkConfig();
  }, [ setFreshForkConfig ]);

  const handleTabChange = useCallback(key => {
    setTabKey(key);
    history.replace(key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`);
  }, [ basePath, history ]);

  const showForkModal = useCallback((): void => {
    setForkModalVisible(true);
  }, [ setForkModalVisible ]);

  const stopPolling = usePolling(fetchExperimentDetails);

  useEffect(() => {
    if (tab && (!TAB_KEYS.includes(tab) || tab === DEFAULT_TAB_KEY)) {
      history.replace(basePath);
    }
  }, [ basePath, history, tab ]);

  useEffect(() => {
    if (experimentDetails.data && terminalRunStates.has(experimentDetails.data.state)) {
      stopPolling();
    }
  }, [ experimentDetails.data, stopPolling ]);

  useEffect(() => {
    return () => source.cancel();
  }, [ source ]);

  useEffect(() => {
    try {
      setFreshForkConfig();
    } catch (e) {
      handleError({
        error: e,
        message: 'failed to load experiment config',
        type: ErrorType.ApiBadResponse,
      });
      setForkModalConfig('failed to load experiment config');
    }
  }, [ setFreshForkConfig ]);

  if (isNaN(id)) {
    return <Message title={`Invalid Experiment ID ${experimentId}`} />;
  } else if (experimentDetails.error) {
    const message = isNotFound(experimentDetails.error) ?
      `Unable to find Experiment ${experimentId}` :
      `Unable to fetch Experiment ${experimentId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!experiment) {
    return <Spinner />;
  }

  return (
    <Page
      breadcrumb={[
        { breadcrumbName: 'Experiments', path: '/experiments' },
        {
          breadcrumbName: `Experiment ${experimentId}`,
          path: `/experiments/${experimentId}`,
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
      <CreateExperimentModal
        config={forkModalConfig}
        okText="Fork"
        parentId={id}
        title={`Fork Experiment ${id}`}
        visible={forkModalVisible}
        onCancel={handleForkModalCancel}
        onConfigChange={setForkModalConfig}
        onVisibleChange={setForkModalVisible} />
    </Page>
  );
};

export default ExperimentDetails;
