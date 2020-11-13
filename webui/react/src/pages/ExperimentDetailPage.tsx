import { Space, Tabs } from 'antd';
import axios from 'axios';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router';

import Badge, { BadgeType } from 'components/Badge';
import CreateExperimentModal from 'components/CreateExperimentModal';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import ExperimentActions from 'pages/ExperimentDetails/ExperimentActions';
import { getExperimentDetails, isNotFound } from 'services/api';
import { ApiState } from 'services/types';
import { ExperimentDetails } from 'types';
import { clone } from 'utils/data';
import { terminalRunStates, upgradeConfig } from 'utils/types';

import css from './ExperimentDetails.module.scss';
import ExperimentOverview from './ExperimentDetails/ExperimentOverview';

const { TabPane } = Tabs;

interface Params {
  experimentId: string;
}

const ExperimentDetailPage: React.FC = () => {
  const { experimentId } = useParams<Params>();
  const id = parseInt(experimentId);
  const [ forkModalVisible, setForkModalVisible ] = useState(false);
  const [ forkModalConfig, setForkModalConfig ] = useState('Loading');
  const [ experimentDetails, setExperimentDetails ] = useState<ApiState<ExperimentDetails>>({
    data: undefined,
    error: undefined,
    isLoading: true,
    source: axios.CancelToken.source(),
  });

  const experiment = experimentDetails.data;

  const fetchExperimentDetails = useCallback(async () => {
    try {
      const response = await getExperimentDetails({
        cancelToken: experimentDetails.source?.token,
        id,
      });
      setExperimentDetails(prev => ({ ...prev, data: response, isLoading: false }));
    } catch (e) {
      if (!experimentDetails.error && !axios.isCancel(e)) {
        setExperimentDetails(prev => ({ ...prev, error: e }));
      }
    }
  }, [ id, experimentDetails.error, experimentDetails.source ]);

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

  const showForkModal = useCallback((): void => {
    setForkModalVisible(true);
  }, [ setForkModalVisible ]);

  const stopPolling = usePolling(fetchExperimentDetails);
  useEffect(() => {
    if (experimentDetails.data && terminalRunStates.has(experimentDetails.data.state)) {
      stopPolling();
    }
  }, [ experimentDetails.data, stopPolling ]);

  useEffect(() => {
    return () => experimentDetails.source?.cancel();
  }, [ experimentDetails.source ]);

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
      backPath={'/experiments'}
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
      showDivider
      subTitle={<Space align="center" size="small">
        {experiment?.config.description}
        <Badge state={experiment.state} type={BadgeType.State} />
        {experiment.archived && <Badge>ARCHIVED</Badge>}
      </Space>}
      title={`Experiment ${experimentId}`}>
      <Tabs className={css.base} defaultActiveKey="overview">
        <TabPane key="overview" tab="Overview">
          <ExperimentOverview experiment={experiment} onTagsChange={fetchExperimentDetails} />
        </TabPane>
        <TabPane key="visualization" tab="Visualization">
          Visualization
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

export default ExperimentDetailPage;
