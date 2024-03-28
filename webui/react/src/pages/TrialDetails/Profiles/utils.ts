import uPlot from 'uplot';

import { humanReadableBytes } from 'utils/string';

// key should be lowercase to match the metric name
const MetricNameUnit = {
  cpu_util_simple: '%',
  disk_iops: 'operations/second',
  disk_throughput_read: 'bytes/second',
  disk_throughput_write: 'bytes/second',
  gpu_free_memory: 'bytes',
  gpu_util: '%',
  memory_free: 'bytes',
  net_throughput_recv: 'bytes/second',
  net_throughput_sent: 'bytes/second',
} as const;

export const getUnitForMetricName = (metricName: string): string => {
  return metricName in MetricNameUnit
    ? MetricNameUnit[metricName as keyof typeof MetricNameUnit]
    : metricName;
};

export const getByteTickValues: uPlot.Axis['values'] = (_self, rawValue) => {
  return rawValue.map(humanReadableBytes);
};

export const getScientificNotationTickValues: uPlot.Axis['values'] = (_self, rawValue) => {
  return rawValue.map((val) => {
    if (val === 0) return val;
    return val > 9_999 || val < -9_999 || (0 < val && val < 0.0001) || (-0.0001 < val && val < 0)
      ? val.toExponential(2)
      : val;
  });
};
