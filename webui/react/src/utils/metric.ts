import { Metric, MetricType, RecordKey, WorkloadGroup } from 'types';
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
  metric: Metric,
): number | undefined => {
  return (
    extractMetricValue(workload, metric) ??
    extractMetricValue(workload, { ...metric, group: MetricType.Validation }) ??
    extractMetricValue(workload, { ...metric, group: MetricType.Training })
  );
};

export const extractMetricValue = (workload: WorkloadGroup, metric: Metric): number | undefined => {
  const source = workload.metrics?.[metric.group]?.metrics ?? {};
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
  /**
   * TODO - also see `src/components/MetricBadgeTag.tsx'
   * Metric group may sometimes end up being `undefined` when an old metric setting
   * is restored and the UI attempts to use it. Adding a safeguard for now.
   * Better approach of hunting down all the places it can be stored as a setting
   * and validating it upon loading and discarding it if invalid.
   */
  const label = !metric.group
    ? metric.name
    : [metric.group, metric.name].join(METRIC_KEY_DELIMITER);
  return label.length > truncateLimit ? label.substring(0, truncateLimit) + '...' : label;
};

export const metricToKey = (metric: Metric): string => {
  try {
    return JSON.stringify(metric, Object.keys(metric).sort());
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

export const metricKeyToName = (key: string): string => metricKeyToMetric(key).name;

export const metricKeyToStr = (key: string): string => {
  const metric = metricKeyToMetric(key);
  return metric ? metricToStr(metric) : '';
};
