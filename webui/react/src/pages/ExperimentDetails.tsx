import { Breadcrumb, Space } from 'antd';
import React from 'react';
import { useParams } from 'react-router';

import Icon from 'components/Icon';
import Link from 'components/Link';
import Message from 'components/Message';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import usePolling from 'hooks/usePolling';
import { useRestApiSimple } from 'hooks/useRestApi';
import { getExperimentDetails, isNotFound } from 'services/api';
import { ExperimentDetailsParams } from 'services/types';
import { ExperimentDetails } from 'types';

interface Params {
  experimentId: string;
}

const ExperimentDetailsComp: React.FC = () => {
  const { experimentId } = useParams<Params>();
  const [ experiment, requestExperimentDetails ] =
  useRestApiSimple<ExperimentDetailsParams, ExperimentDetails>(
    getExperimentDetails, { id: parseInt(experimentId) });
  usePolling(() => requestExperimentDetails);

  if (experiment.error !== undefined) {
    const message = isNotFound(experiment.error) ? `Experiment ${experimentId} not found.`
      : `Failed to fetch experiment ${experimentId}.`;
    return (
      <Page hideTitle title="Not Found">
        <Message>{message}</Message>
      </Page>
    );
  }

  if (!experiment.data || experiment.isLoading) {
    <Spinner fillContainer />;
  }

  return (
    <Page title={`Experiment ${experiment.data?.config.description}`}>
      <Breadcrumb>
        <Breadcrumb.Item>
          <Space align="center" size="small">
            <Icon name="experiment" size="small" />
            <Link path="/det/experiments">Experiments</Link>
          </Space>
        </Breadcrumb.Item>
        <Breadcrumb.Item>
          <span>{experimentId}</span>
        </Breadcrumb.Item>
      </Breadcrumb>
    </Page>
  );
};

export default ExperimentDetailsComp;
