import { Breadcrumb, Space } from 'antd';
import React, { useCallback } from 'react';
import { useParams } from 'react-router';

import TrialActions from 'components/TrialActions';
import TrialInfoBox from 'components/TrialInfoBox';
import Icon from 'components/Icon';
import Link from 'components/Link';
import Message from 'components/Message';
import Page from 'components/Page';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import usePolling from 'hooks/usePolling';
import { useRestApiSimple } from 'hooks/useRestApi';
import { getTrialDetails, isNotFound } from 'services/api';
import { TrialDetailsParams } from 'services/types';
import { TrialDetails } from 'types';

interface Params {
  trialId: string;
}

const TrialDetailsComp: React.FC = () => {
  const { trialId: trialIdParam } = useParams<Params>();
  const trialId = parseInt(trialIdParam);
  const [ trial, setExpRequestParams ] =
  useRestApiSimple<TrialDetailsParams, TrialDetails>(
    getTrialDetails, { id: trialId });
  const pollTrialDetails = useCallback(() => setExpRequestParams({ id: trialId }),
    [ setExpRequestParams, trialId ]);
  usePolling(pollTrialDetails);

  if (isNaN(trialId)) {
    return (
      <Page hideTitle title="Not Found">
        <Message>Bad trial ID {trialIdParam}</Message>
      </Page>
    );
  }

  if (trial.error !== undefined) {
    const message = isNotFound(trial.error) ? `Trial ${trialId} not found.`
      : `Failed to fetch trial ${trialId}.`;
    return (
      <Page hideTitle title="Not Found">
        <Message>{message}</Message>
      </Page>
    );
  }

  if (!trial.data) {
    return <Spinner fillContainer />;
  }

  return (
    <Page title={`Trial ${trial.data?.config.description}`}>
      <Breadcrumb>
        <Breadcrumb.Item>
          <Space align="center" size="small">
            <Icon name="trial" size="small" />
            <Link path="/det/trials">Trials</Link>
          </Space>
        </Breadcrumb.Item>
        <Breadcrumb.Item>
          <span>{trialId}</span>
        </Breadcrumb.Item>
      </Breadcrumb>
      <TrialActions trial={trial.data} onSettled={pollTrialDetails} />
      <TrialInfoBox trial={trial.data} />
      <Section title="Chart" />
      <Section title="Trials" />

    </Page>
  );
};

export default TrialDetailsComp;
