import { OldMetric, RecordKey } from 'types';
import { Metric, MetricType, WorkloadGroup } from 'types';
import { alphaNumericSorter } from 'utils/sort';

export const METRIC_KEY_DELIMITER = '.';

/**
 * Metrics are sorted by their type first (alphabetically) followed by their name (alphabetically).
 */
export const metricSorter = (a: Metric, b: Metric): number => {
  if (a.group !== b.group) return alphaNumericSorter(a.group, b.group);
  return alphaNumericSorter(a.name, b.name);
};

export const extractMetrics = (workloads: WorkloadGroup[]): Metric[] => {
  const trainingNames = workloads
    .filter((workload) => workload.metrics.training?.metrics)
    .reduce((acc, workload) => {
      Object.keys(workload.metrics.training?.metrics as Record<string, number>).forEach((name) => {
        acc.add(name);
      });
      return acc;
    }, new Set<string>());

  const trainingMetrics: Metric[] = Array.from(trainingNames).map((name) => {
    return { group: MetricType.Training, name };
  });

  const validationNames = workloads
    .filter((workload) => workload.metrics.validation?.metrics)
    .reduce((acc, workload) => {
      Object.keys(workload.metrics.validation?.metrics as Record<string, number>).forEach(
        (name) => {
          acc.add(name);
        },
      );
      return acc;
    }, new Set<string>()) as Set<string>;

  const validationMetrics: Metric[] = Array.from(validationNames).map((name) => {
    return { group: MetricType.Validation, name };
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
  const source = workload.metrics[metric.type]?.metrics ?? {};
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
  const group = metric.group.substring(0, 1).toUpperCase();
  const name =
    metric.name.length > truncateLimit
      ? metric.name.substring(0, truncateLimit) + '...'
      : metric.name;
  return `[${group}] ${name}`;
};

export const metricToOldMetric = (metric: Metric): OldMetric => {
  const type = metric.group === MetricType.Training ? MetricType.Training : MetricType.Validation;
  return { name: metric.name, type };
};

export const oldMetricToMetric = (oldMetric: OldMetric): Metric => {
  return { group: oldMetric.type, name: oldMetric.name };
};

export const metricToKey = (metric: Metric): string => {
  try {
    return JSON.stringify(metric);
  } catch (e) {
    return [metric.group, metric.name].join(METRIC_KEY_DELIMITER);
  }
};

export const metricKeyToMetric = (value: string): Metric => {
  try {
    return JSON.parse(value);
  } catch (e) {
    const parts = value.split(METRIC_KEY_DELIMITER);
    return parts.length < 2
      ? { group: parts[0] ?? 'NO_GROUP', name: value }
      : { group: parts[0], name: parts.slice(1).join(METRIC_KEY_DELIMITER) };
  }
};

export const metricKeyToOldMetric = (value: string): OldMetric | undefined => {
  const parts = value.split('|');
  if (parts.length !== 2) return;
  if (![MetricType.Training, MetricType.Validation].includes(parts[0] as MetricType)) return;
  return { name: parts[1], type: parts[0] as MetricType };
};

export const metricKeyToName = (key: string): string => metricKeyToMetric(key).name;

export const metricKeyToType = (key: string): MetricType | undefined =>
  metricKeyToOldMetric(key)?.type;

export const metricKeyToStr = (key: string): string => {
  const metric = metricKeyToMetric(key);
  return metric ? metricToStr(metric) : '';
};
