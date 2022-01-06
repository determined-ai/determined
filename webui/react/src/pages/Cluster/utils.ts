import { V1ResourceAllocationAggregatedEntry } from 'services/api-ts-sdk';
import { sumArrays } from 'utils/array';
import { secondToHour } from 'utils/datetime';

import { GroupBy } from './ClusterHistoricalUsage.settings';

export interface ResourceAllocationChartSeries {
  groupedBy: GroupBy,
  hoursByAgentLabel: Record<string, number[]>,
  hoursByExperimentLabel: Record<string, number[]>,
  hoursByResourcePool: Record<string, number[]>,
  hoursByUsername: Record<string, number[]>,
  hoursTotal: Record<string, number[]>,
  time: string[],
}

export const mapResourceAllocationApiToChartSeries = (
  apiRes: Array<V1ResourceAllocationAggregatedEntry>,
  grouping: GroupBy,
): ResourceAllocationChartSeries => {
  return {
    groupedBy: grouping,
    hoursByAgentLabel: mapToChartSeries(apiRes.map(item => item.byAgentLabel)),
    hoursByExperimentLabel: mapToChartSeries(apiRes.map(item => item.byExperimentLabel)),
    hoursByResourcePool: mapToChartSeries(apiRes.map(item => item.byResourcePool)),
    hoursByUsername: mapToChartSeries(apiRes.map(item => item.byUsername)),
    hoursTotal: { total: apiRes.map(item => secondToHour(item.seconds)) },
    time: apiRes.map(item => item.periodStart),
  };
};

const mapToChartSeries = (labelByPeriod: Record<string, number>[]): Record<string, number[]> => {
  // 1. convert [periodIndex: {tag: seconds}, ...] to {tag: {periodIndex: hours}, ...}
  const periodByLabelIndexed: Record<string, Record<number, number>> = {};
  labelByPeriod.forEach((period, periodIndex) => {
    Object.keys(period).forEach(tag => {
      periodByLabelIndexed[tag] = {
        ...(periodByLabelIndexed[tag] || {}),
        [periodIndex]: secondToHour(period[tag]),
      };
    });
  });

  // 2. convert {tag: {periodIndex: hours}, ...} to {tag: [hours, ...], ...}
  const periodByLabelIndexedFlat: Record<string, number[]> = {};
  Object.keys(periodByLabelIndexed).forEach(tag => {
    periodByLabelIndexedFlat[tag] = [];
    for (let i = 0; i < labelByPeriod.length; i++) {
      periodByLabelIndexedFlat[tag].push(periodByLabelIndexed[tag][i] || 0);
    }
  });

  // 3. find top 5 labels
  const topLabels = Object.keys(periodByLabelIndexedFlat).map(tag => {
    const hours = periodByLabelIndexedFlat[tag].reduce((acc, val) => acc + val, 0);
    return [ tag, hours ];
  })
    .sort((a, b) => ((b[1] as number) - (a[1] as number)))
    .slice(0, 5)
    .map(item => item[0]);

  // 4. sum non-top labels hours into "other labels"
  let ret = {};
  let otherLabels: number[] = [];
  Object.keys(periodByLabelIndexedFlat).forEach(tag => {
    if (topLabels.includes(tag)) {
      ret = { ...ret, [tag]: periodByLabelIndexedFlat[tag] };
    } else {
      otherLabels = sumArrays(otherLabels, periodByLabelIndexedFlat[tag]);
    }
  });
  if (otherLabels.length > 0) {
    ret = { ...ret, ['other labels']: otherLabels };
  }

  return ret;
};
