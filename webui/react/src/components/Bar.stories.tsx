import React from 'react';

import Bar, { Props } from './Bar';

export default {
  component: Bar,
  title: 'Bar',
};

const Wrapper: React.FC<Props> = props => (
  <div style={{ width: 240 }}>
    <Bar {...props} />
  </div>
);

export const Default = (): React.ReactNode => <Wrapper parts={[
  {
    color: 'red',
    label: 'labelA',
    percent: 0.3,
  },
  {
    color: 'blue',
    label: 'labelB',
    percent: 0.2,
  },
  {
    color: 'yellow',
    label: 'labelc',
    percent: 0.5,
  },
]} />;
