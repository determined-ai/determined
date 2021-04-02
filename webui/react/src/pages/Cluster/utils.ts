import { V1ResourceAllocationAggregatedEntry } from 'services/api-ts-sdk';
import { sumArrays } from 'utils/array';
import { secondToHour } from 'utils/time';

import { GroupBy } from './ClusterHistoricalUsage';

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
  // 1. convert [periodIndex: {label: seconds}, ...] to {label: {periodIndex: hours}, ...}
  const periodByLabelIndexed: Record<string, Record<number, number>> = {};
  labelByPeriod.forEach((period, periodIndex) => {
    Object.keys(period).forEach(label => {
      periodByLabelIndexed[label] = {
        ...(periodByLabelIndexed[label] || {}),
        [periodIndex]: secondToHour(period[label]),
      };
    });
  });

  // 2. convert {label: {periodIndex: hours}, ...} to {label: [hours, ...], ...}
  const periodByLabelIndexedFlat: Record<string, number[]> = {};
  Object.keys(periodByLabelIndexed).forEach(label => {
    periodByLabelIndexedFlat[label] = [];
    for (let i = 0; i < labelByPeriod.length; i++) {
      periodByLabelIndexedFlat[label].push(periodByLabelIndexed[label][i] || 0);
    }
  });

  // 3. find top 5 labels
  const topLabels = Object.keys(periodByLabelIndexedFlat).map(label => {
    const hours = periodByLabelIndexedFlat[label].reduce((acc, val) => acc + val, 0);
    return [ label, hours ];
  })
    .sort((a, b) => ((b[1] as number) - (a[1] as number)))
    .slice(0, 5)
    .map(item => item[0]);

  // 4. sum non-top labels hours into "other labels"
  let ret = {};
  let otherLabels: number[] = [];
  Object.keys(periodByLabelIndexedFlat).forEach(label => {
    if (topLabels.includes(label)) {
      ret = { ...ret, [label]: periodByLabelIndexedFlat[label] };
    } else {
      otherLabels = sumArrays(otherLabels, periodByLabelIndexedFlat[label]);
    }
  });
  if (otherLabels.length > 0) {
    ret = { ...ret, ['other labels']: otherLabels };
  }

  return ret;
};
