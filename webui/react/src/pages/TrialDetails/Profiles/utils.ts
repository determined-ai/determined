export const getUnitForMetricName = (metricName: string): string => {
  if (metricName === 'cpu_util_simple') return '%';
  if (metricName === 'disk_throughput_read') return 'bytes/second';
  if (metricName === 'disk_throughput_write') return 'bytes/second';
  if (metricName === 'free_memory') return 'Gigabytes';
  if (metricName === 'gpu_util') return '%';
  if (metricName === 'net_throughput_recv') return 'Gigabit/s';
  if (metricName === 'net_throughput_sent') return 'Gigabit/s';
  if (metricName === 'samples_per_second') return 'Samples/s';
  if (metricName === 'gpu_free_memory') return 'Bytes';
  if (metricName === 'disk_iops') return 'Bytes/s';
  return metricName;
};
