import { useState } from 'react';

import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import { Scale } from 'types';

export const useLineChart = (): {
  scale: Scale;
  setScale: React.Dispatch<React.SetStateAction<Scale>>;
  setXAxis: React.Dispatch<React.SetStateAction<XAxisDomain>>;
  xAxis: XAxisDomain;
} => {
  const [xAxis, setXAxis] = useState<XAxisDomain>(XAxisDomain.Batches);
  const [scale, setScale] = useState<Scale>(Scale.Linear);
  return { scale, setScale, setXAxis, xAxis };
};
