import { OldMetric, RecordKey } from 'types';
import { Metric, MetricType, WorkloadGroup } from 'types';

import { alphaNumericSorter } from '../utils/sort';

/**
 * Metrics are sorted by their type first (alphabetically) followed by their name (alphabetically).
 */
export const metricSorter = (a: Metric, b: Metric): number => {
  if (a.type !== b.type) return alphaNumericSorter(a.type, b.type);
  return alphaNumericSorter(a.name, b.name);
};

export const extractMetrics = (workloads: WorkloadGroup[]): Metric[] => {
  const trainingNames = workloads
    .filter((workload) => workload.training?.metrics)
    .reduce((acc, workload) => {
      Object.keys(workload.training?.metrics as Record<string, number>).forEach((name) => {
        acc.add(name);
      });
      return acc;
    }, new Set<string>());

  const trainingMetrics: Metric[] = Array.from(trainingNames).map((name) => {
    return { name, type: MetricType.Training };
  });

  const validationNames = workloads
    .filter((workload) => workload.validation?.metrics)
    .reduce((acc, workload) => {
      Object.keys(workload.validation?.metrics as Record<string, number>).forEach((name) => {
        acc.add(name);
      });
      return acc;
    }, new Set<string>()) as Set<string>;

  const validationMetrics: Metric[] = Array.from(validationNames).map((name) => {
    return { name, type: MetricType.Validation };
  });

  return [...validationMetrics, ...trainingMetrics].sort(metricSorter);
};

export const extractMetricSortValue = (
  workload: WorkloadGroup,
  metric: OldMetric,
): number | undefined => {
  return (
    extractMetricValue(workload, metric) ??
    extractMetricValue(workload, { ...metric, type: MetricType.Validation }) ??
    extractMetricValue(workload, { ...metric, type: MetricType.Training })
  );
};

export const extractMetricValue = (
  workload: WorkloadGroup,
  metric: OldMetric,
): number | undefined => {
  const source = workload[metric.type]?.metrics ?? {};
  return source[metric.name];
};

export const getMetricValue = (
  workload?: { metrics?: Record<RecordKey, number> },
  metric?: string,
): number | undefined => {
  if (!metric || !workload?.metrics) return undefined;
  return workload?.metrics[metric];
};

export const isMetric = (metric?: Metric): metric is Metric => metric !== undefined;
export const metricToStr = (metric: Metric, truncateLimit = 30): string => {
  const type = metric.type === MetricType.Training ? 'T' : 'V';
  const name =
    metric.name.length > truncateLimit
      ? metric.name.substring(0, truncateLimit) + '...'
      : metric.name;
  return `[${type}] ${name}`;
};

export const metricToKey = (metric: Metric): string => {
  return `${metric.type}/${metric.name}`;
};

export const metricKeyToMetric = (value: string): Metric | undefined => {
  const parts = value.split('/');
  if (parts.length !== 2) return;
  return { name: parts[1], type: parts[0] };
};

export const metricKeyToOldMetric = (value: string): OldMetric | undefined => {
  const parts = value.split('|');
  if (parts.length !== 2) return;
  if (![MetricType.Training, MetricType.Validation].includes(parts[0] as MetricType)) return;
  return { name: parts[1], type: parts[0] as MetricType };
};

export const metricKeyToName = (key: string): string => metricKeyToMetric(key)?.name ?? '';

export const metricKeyToType = (key: string): MetricType | undefined =>
  metricKeyToOldMetric(key)?.type;

export const metricKeyToStr = (key: string): string => {
  const metric = metricKeyToMetric(key);
  return metric ? metricToStr(metric) : '';
};
