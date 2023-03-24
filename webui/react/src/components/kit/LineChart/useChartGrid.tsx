import React, { useState } from 'react';

import { ChartGrid, GroupProps } from 'components/kit/LineChart';
import { Scale } from 'types';

export const useChartGrid = (): ((
  props: Omit<GroupProps, 'scale' | 'setScale'>,
) => JSX.Element) => {
  const [scale, setScale] = useState<Scale>(Scale.Linear);
  return (props) => <ChartGrid {...props} scale={scale} setScale={setScale} />;
};
