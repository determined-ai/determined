import { Breadcrumb, Space } from 'antd';
import React from 'react';

import Icon from 'components/Icon';
import Link from 'components/Link';

export default {
  component: Breadcrumb,
  title: 'Breadcrumb',
};

export const Default = (): React.ReactNode => (
  <Breadcrumb>
    <Breadcrumb.Item>
      <Space align="center" size="small">
        <Icon name="experiment" size="small" />
        <Link path="/experiments">Experiments</Link>
      </Space>
    </Breadcrumb.Item>
    <Breadcrumb.Item>
      <Link path="/experiments/3">Experiment 3</Link>
    </Breadcrumb.Item>
  </Breadcrumb>
);
export const TrialDetail = (): React.ReactNode => (
  <Breadcrumb>
    <Breadcrumb.Item>
      <Space align="center" size="small">
        <Icon name="experiment" size="small" />
        <Link path="/experiments">Experiments</Link>
      </Space>
    </Breadcrumb.Item>
    <Breadcrumb.Item>
      <Link path="/experiments/3"> Experiment 3</Link>
    </Breadcrumb.Item>
    <Breadcrumb.Item>
      <Link path="/experiments/3/trials/34">Trial 34</Link>
    </Breadcrumb.Item>
  </Breadcrumb>
);
