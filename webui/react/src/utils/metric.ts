import { MetricName, MetricType, RecordKey, WorkloadGroup } from 'types';
import { metricNameSorter } from 'utils/sort';

export const extractMetricNames = (workloads: WorkloadGroup[]): MetricName[] => {
  const trainingNames = workloads
    .filter(workload => workload.training?.metrics)
    .reduce((acc, workload) => {
      Object.keys(workload.training?.metrics as Record<string, number>).forEach(name => {
        acc.add(name);
      });
      return acc;
    }, new Set<string>());

  const trainingMetrics: MetricName[] = Array.from(trainingNames).map(name => {
    return { name, type: MetricType.Training };
  });

  const validationNames = workloads
    .filter(workload => workload.validation?.metrics)
    .reduce((acc, workload) => {
      Object.keys(workload.validation?.metrics as Record<string, number>).forEach(name => {
        acc.add(name);
      });
      return acc;
    }, new Set<string>()) as Set<string>;

  const validationMetrics: MetricName[] = Array.from(validationNames).map(name => {
    return { name, type: MetricType.Validation };
  });

  return [ ...validationMetrics, ...trainingMetrics ].sort(metricNameSorter);
};

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
