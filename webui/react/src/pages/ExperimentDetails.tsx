import { Breadcrumb, Space } from 'antd';
import React, { useCallback } from 'react';
import { useParams } from 'react-router';

import ExperimentActions from 'components/ExperimentActions';
import ExperimentInfoBox from 'components/ExperimentInfoBox';
import Icon from 'components/Icon';
import Link from 'components/Link';
import Message from 'components/Message';
import Page from 'components/Page';
import Section from 'components/Section';
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
  const { experimentId: experimentIdParam } = useParams<Params>();
  const experimentId = parseInt(experimentIdParam);
  const [ experiment, setExpRequestParams ] =
  useRestApiSimple<ExperimentDetailsParams, ExperimentDetails>(
    getExperimentDetails, { id: experimentId });
  const pollExperimentDetails = useCallback(() => setExpRequestParams({ id: experimentId }),
    [ setExpRequestParams, experimentId ]);
  usePolling(pollExperimentDetails);

  if (isNaN(experimentId)) {
    return (
      <Page hideTitle title="Not Found">
        <Message>Bad experiment ID {experimentIdParam}</Message>
      </Page>
    );
  }

  if (experiment.error !== undefined) {
    const message = isNotFound(experiment.error) ? `Experiment ${experimentId} not found.`
      : `Failed to fetch experiment ${experimentId}.`;
    return (
      <Page hideTitle title="Not Found">
        <Message>{message}</Message>
      </Page>
    );
  }

  if (!experiment.data) {
    return <Spinner fillContainer />;
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
      <ExperimentActions experiment={experiment.data} onSettled={pollExperimentDetails} />
      <ExperimentInfoBox experiment={experiment.data} />
      <Section title="Chart" />
      <Section title="Trials" />

    </Page>
  );
};

export default ExperimentDetailsComp;
