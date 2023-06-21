import React, { useMemo } from 'react';

import { DURATION_UNIT_MEASURES, durationInEnglish } from 'utils/datetime';

interface Props {
  duration: number;
  units?: number;
}

const TimeDuration: React.FC<Props> = ({ duration, units = 2 }: Props) => {
  const durationString = useMemo(() => {
    const options = {
      conjunction: ' ',
      delimiter: ' ',
      largest: units,
      serialComma: false,
      unitMeasures: { ...DURATION_UNIT_MEASURES, ms: 1000 },
    };
    return durationInEnglish(duration, options);
  }, [duration, units]);

  return <div>{durationString}</div>;
};

export default TimeDuration;
