import { MetricName, MetricType, WorkloadGroup } from 'types';
import { metricNameSorter } from 'utils/sort';

import { RecordKey } from '../shared/types';

export const extractMetricValue = (
  workload: WorkloadGroup,
  metricName: MetricName,
): number | undefined => {
  const source = workload[metricName.type]?.metrics ?? {};
  return source[metricName.name];
};

export const getMetricValue = (
  workload?: { metrics?: Record<RecordKey, number> },
  metricName?: string,
): number | undefined => {
  if (!metricName || !workload?.metrics) return undefined;
  return workload?.metrics[metricName];
};

export const metricNameToStr = (metricName: MetricName, truncateLimit = 30): string => {
  const type = metricName.type === MetricType.Training ? 'T' : 'V';
  const name = metricName.name.length > truncateLimit ?
    metricName.name.substr(0, truncateLimit) + '...' : metricName.name;
  return `[${type}] ${name}`;
};

export const metricNameToValue = (metricName: MetricName): string => {
  return `${metricName.type}|${metricName.name}`;
};

export const valueToMetricName = (value: string): MetricName | undefined => {
  const parts = value.split('|');
  if (parts.length !== 2) return;
  if (![ MetricType.Training, MetricType.Validation ].includes(parts[0] as MetricType)) return;
  return { name: parts[1], type: parts[0] as MetricType };
};
