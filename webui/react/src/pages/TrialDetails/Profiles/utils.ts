import { AlignedData } from 'uplot';

export const convertMetricsToUplotData = (
  data: Record<number, Record<string, number>>,
  nameList: string[],
): AlignedData => {
  const series: (number | null)[][] = [];
  const timeSerie: number[] = [];

  // Sort time keys are not guaranteed to arrive in order so we sort them first.
  const timeKeys = Object.keys(data).map(k => parseInt(k)).sort();

  for (const key of timeKeys) {
    const list = data[key];

    timeSerie.push(key);
    nameList.forEach((name, nameIndex) => {
      if (!series[nameIndex]) { series[nameIndex] = []; }
      series[nameIndex].push(name in list ? list[name] : null);
    });
  }

  return [ timeSerie, ...series ];
};

export const getUnitForMetricName = (metricName: string): string => {
  if (metricName === 'cpu_util_simple') return '%';
  if (metricName === 'disk_throughput_read') return 'bytes/second';
  if (metricName === 'disk_throughput_write') return 'bytes/second';
  if (metricName === 'free_memory') return 'Gigabytes';
  if (metricName === 'gpu_util') return '%';
  if (metricName === 'net_throughput_recv') return 'Gigabit/s';
  if (metricName === 'net_throughput_sent') return 'Gigabit/s';
  if (metricName === 'samples_per_second') return 'Samples/s';
  return metricName;
};
