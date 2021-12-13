import { Tooltip } from 'antd';
import React from 'react';

import OverviewStats from 'components/OverviewStats';
import { RPStats } from 'types';

interface Props {
  focused?: boolean;
  onClick?: () => void;
  stats: RPStats
}

const RPStatsOverview: React.FC<Props> = ({ focused, onClick, stats }) => {
  return (
    <OverviewStats
      focused={focused}
      title={stats.resourcePool}
      onClick={onClick}>
      <Tooltip title="Scheduled Jobs">
        {stats.stats.scheduledCount}
      </Tooltip>{' / '}
      <Tooltip title="All Jobs">
        {stats.stats.queuedCount + stats.stats.scheduledCount}
      </Tooltip>
    </OverviewStats>
  );
};

export default RPStatsOverview;
