import uPlot from 'uplot';

// key should be lowercase to match the metric name
const MetricNameUnit = {
  cpu_util_simple: '%',
  disk_iops: 'Bytes/s',
  disk_throughput_read: 'bytes/second',
  disk_throughput_write: 'bytes/second',
  free_memory: 'Gigabytes',
  gpu_free_memory: 'Bytes',
  gpu_util: '%',
  net_throughput_recv: 'Gigabit/s',
  net_throughput_sent: 'Gigabit/s',
  samples_per_second: 'Samples/s',
} as const;

export const getUnitForMetricName = (metricName: string): string => {
  return metricName in MetricNameUnit
    ? MetricNameUnit[metricName as keyof typeof MetricNameUnit]
    : metricName;
};

export const getScientificNotationTickValues: uPlot.Axis['values'] = (_self, rawValue) => {
  return rawValue.map((val) => {
    if (val === 0) return val;
    return val > 9_999 || val < -9_999 || (0 < val && val < 0.0001) || (-0.0001 < val && val < 0)
      ? val.toExponential(2)
      : val;
  });
};
