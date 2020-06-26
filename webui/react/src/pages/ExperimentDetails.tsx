import { Breadcrumb, Space } from 'antd';
import React from 'react';
import { useParams } from 'react-router';

import Icon from 'components/Icon';
import Link from 'components/Link';
import Page from 'components/Page';

interface Params {
  experimentId: string;
}

const ExperimentDetails: React.FC = () => {
  const { experimentId } = useParams<Params>();
  return (
    <Page title={`Experiment ${experimentId}`}>
      <Breadcrumb>
        <Breadcrumb.Item>
          <Space align="center" size="small">
            <Icon name="experiment" size="small" />
            <Link path="/det/experiments">Experiments</Link>
          </Space>
        </Breadcrumb.Item>
        <Breadcrumb.Item>
          <span>3</span>
        </Breadcrumb.Item>
      </Breadcrumb>
    </Page>
  );
};

export default ExperimentDetails;
