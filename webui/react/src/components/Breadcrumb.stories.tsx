import { Breadcrumb, Space } from 'antd';
import React from 'react';

import Link from 'components/Link';
import { paths } from 'routes/utils';
import Icon from 'shared/components/Icon/Icon';

export default {
  component: Breadcrumb,
  title: 'Breadcrumb',
};

export const Default = (): React.ReactNode => (
  <Breadcrumb>
    <Breadcrumb.Item>
      <Space align="center" size="small">
        <Icon name="experiment" size="small" />
        <Link path={paths.experimentList()}>Experiments</Link>
      </Space>
    </Breadcrumb.Item>
    <Breadcrumb.Item>
      <Link path={paths.experimentDetails(3)}>Experiment 3</Link>
    </Breadcrumb.Item>
  </Breadcrumb>
);
export const TrialDetail = (): React.ReactNode => (
  <Breadcrumb>
    <Breadcrumb.Item>
      <Space align="center" size="small">
        <Icon name="experiment" size="small" />
        <Link path={paths.experimentList()}>Experiments</Link>
      </Space>
    </Breadcrumb.Item>
    <Breadcrumb.Item>
      <Link path={paths.experimentDetails(3)}> Experiment 3</Link>
    </Breadcrumb.Item>
    <Breadcrumb.Item>
      <Link path={paths.trialDetails(34, 3)}>Trial 34</Link>
    </Breadcrumb.Item>
  </Breadcrumb>
);
