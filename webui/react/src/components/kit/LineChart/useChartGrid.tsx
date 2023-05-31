import React, { useState } from 'react';

import { Scale } from 'components/kit/internal/types';
import { ChartGrid, GroupProps } from 'components/kit/LineChart';

export const useChartGrid = (): ((
  props: Omit<GroupProps, 'scale' | 'setScale'>,
) => JSX.Element) => {
  const [scale, setScale] = useState<Scale>(Scale.Linear);
  return (props) => <ChartGrid {...props} scale={scale} setScale={setScale} />;
};
