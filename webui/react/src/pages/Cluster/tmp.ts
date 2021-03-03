import { Dayjs } from 'dayjs';

import { GroupBy, UsagePeriod } from './ClusterHistoricalUsage';

// todo: when removing this can un-export interfaces UsagePeriod

// src: https://stackoverflow.com/a/10305424
const generateFakeHours = (num: number, ret: number[]): number[] => {
  if (num <= 0) {
    return ret;
  }

  const newNum = Math.ceil((Math.random() * num) / 2);
  return generateFakeHours(num - newNum, [ ...ret, newNum ]);
};

const generateFakeLabelHours = (hoursTotal: number): Record<string, number> => {
  return generateFakeHours(hoursTotal, []).reduce((acc, curr, index) => {
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    acc['label ' + index] = curr;
    return acc;
  }, {});
};

export const generateFakeUsagePeriod = (
  groupBy: GroupBy,
  afterDate: Dayjs,
  beforeDate: Dayjs,
): UsagePeriod[] => {
  const dataPoint = beforeDate.diff(afterDate, groupBy) + 1;
  const dateFormat = 'YYYY-MM' + (groupBy === GroupBy.Day ? '-DD' : '');
  const ret: UsagePeriod[] = [];

  for (let i = 0; i < dataPoint; i++) {
    const currentDate = afterDate.add(i, groupBy);
    const hoursTotal = Math.ceil(Math.random() * 100);
    ret.push({
      hoursByAgentLabel: generateFakeLabelHours(hoursTotal),
      hoursByExperimentLabel: generateFakeLabelHours(hoursTotal),
      hoursByResourcePool: generateFakeLabelHours(hoursTotal),
      hoursByUsername: generateFakeLabelHours(hoursTotal),
      hoursTotal,
      periodStart: currentDate.format(dateFormat),
      periodType: groupBy,
    });
  }

  return ret;
};
