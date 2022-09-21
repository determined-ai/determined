import React from 'react';

import OverviewStats from './OverviewStats';

export default {
  component: OverviewStats,
  title: 'Determined/OverviewStats',
};

export const Default = (): React.ReactNode => (
  <OverviewStats title="stats title">160</OverviewStats>
);
