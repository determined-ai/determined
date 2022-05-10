import dayjs from 'dayjs';
import React, { useMemo, useRef } from 'react';
import uPlot, { AlignedData } from 'uplot';

import Section from 'components/Section';
import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import { useProfilesFilterContext } from 'pages/TrialDetails/Profiles/ProfilesFiltersProvider';
import SystemMetricFilter from 'pages/TrialDetails/Profiles/SystemMetricFilter';
import { convertMetricsToUplotData, getUnitForMetricName } from 'pages/TrialDetails/Profiles/utils';
import { glasbeyColor } from 'shared/utils/color';

import css from './ProfilesEnabled.module.scss';
import { MetricType } from './types';

const CHART_HEIGHT = 300;
const CHART_STYLE: React.CSSProperties = { height: '100%', paddingBottom: 16 };

/*
 * Shared uPlot chart options.
 */
const tzDate = (ts: number) => uPlot.tzDate(new Date(ts * 1e3), 'Etc/UTC');
const matchSyncKeys: uPlot.Cursor.Sync.ScaleKeyMatcher = (own, ext) => own === ext;
const timeAxis: uPlot.Axis = {
  label: 'Time',
  scale: 'x',
  space: (self, axisIdx, scaleMin, scaleMax, plotDim) => {
    const rangeMs = scaleMax - scaleMin;
    const msPerSec = plotDim / rangeMs;
    return Math.max(60, msPerSec * 10 * 1000);
  },
  values: (self, splits) => {
    return splits.map(i => dayjs.utc(i).format('HH:mm:ss'));
  },
};
const metricAxis = (filterName = '') => ({
  label: getUnitForMetricName(filterName),
  scale: 'y',
  size: (self: uPlot, values: string[]) => {
    if (!values) return 50;
    const maxChars = Math.max(...values.map(el => el.toString().length));
    return 25 + Math.max(25, maxChars * 8);
  },
});
const timeSeries: uPlot.Series = {
  label: 'Time',
  scale: 'x',
  value: (self, rawValue) => dayjs.utc(rawValue).format('HH:mm:ss.SSS'),
};
const batchSeries: uPlot.Series = {
  class: css.disabledLegend,
  label: 'Batch',
  scale: 'y',
  show: false,
};
const seriesMapping = (name: string, index: number) => ({
  label: name,
  points: { show: false },
  scale: 'y',
  spanGaps: true,
  stroke: glasbeyColor(index),
  width: 2,
});
const fillerMapping = () => ({ class: css.hiddenLegend, scale: 'y', show: false });

const ProfilesEnabled: React.FC = () => {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartSyncKey = useRef(uPlot.sync('x'));
  const { metrics, settings } = useProfilesFilterContext();

  const chartOptions = useMemo(() => {
    // Define shared options between all charts.
    const sharedOptions: Partial<Options> = {
      cursor: {
        focus: { prox: 16 },
        lock: true,
        sync: {
          key: chartSyncKey.current.key,
          match: [ matchSyncKeys, matchSyncKeys ],
          setSeries: true,
        },
      },
      height: CHART_HEIGHT,
      scales: { x: { time: false } },
      tzDate,
    };

    // Convert metrics into uPlot-friendly data and define chart specific configs.
    const config = {
      [MetricType.System]: { axes: [ timeAxis, metricAxis(settings.name) ] },
      [MetricType.Throughput]: { axes: [ timeAxis, metricAxis('samples_per_second') ] },
      [MetricType.Timing]: { axes: [ timeAxis, { label: 'Seconds', scale: 'y' } ] },
    };

    // Finalize uPlot data and options for all charts.
    const metricKeys = [ MetricType.System, MetricType.Throughput, MetricType.Timing ];
    const uPlotData = metricKeys.reduce((acc, key) => {
      const series = metricKeys.reduce((acc, seriesKey) => {
        const metricNames = metrics[seriesKey].names.slice(1);
        if (seriesKey === key) {
          const series = metricNames.map(seriesMapping);
          acc.push(...series);
        } else {
          const filler = metricNames.map(fillerMapping);
          acc.push(...filler);
        }
        return acc;
      }, [ timeSeries, batchSeries ]);

      const options = { ...sharedOptions, axes: config[key].axes, series } as Options;

      return { ...acc, [key]: options };
    }, {} as Record<MetricType, Options>);

    return uPlotData;
  }, [ settings.name, metrics ]);

  const chartData = useMemo(() => {
    // Convert metrics into uPlot-friendly data and define chart specific configs.
    const metricData = {
      [MetricType.System]: {
        data: convertMetricsToUplotData(
          metrics[MetricType.System].dataByTime,
          metrics[MetricType.System].names,
        ),
      },
      [MetricType.Throughput]: {
        data: convertMetricsToUplotData(
          metrics[MetricType.Throughput].dataByTime,
          metrics[MetricType.Throughput].names,
        ),
      },
      [MetricType.Timing]: {
        data: convertMetricsToUplotData(
          metrics[MetricType.Timing].dataByTime,
          metrics[MetricType.Timing].names,
        ),
      },
    };

    // Finalize uPlot data and options for all charts.
    const metricKeys = Object.keys(metrics) as MetricType[];
    const uPlotData = metricKeys.reduce((acc, key) => {
      const [ times, batches ] = metricData[key].data;

      // Pad the series data with empty series data.
      const data = metricKeys.reduce((acc, seriesKey) => {
        const seriesData = metricData[seriesKey].data.slice(2);
        if (seriesKey === key) {
          acc.push(...seriesData);
        } else {
          const fillerData = new Array(times.length).fill(null) || [];
          const filler = new Array(seriesData.length).fill(fillerData) || [];
          acc.push(...filler);
        }
        return acc;
      }, [ times || [], batches || [] ]);

      // Pad the series config with blank series where applicable.

      return { ...acc, [key]: data };
    }, {} as Record<MetricType, AlignedData>);

    return uPlotData;
  }, [ metrics ]);

  return (
    <div ref={containerRef}>
      <Section
        bodyBorder
        bodyNoPadding
        title="Throughput">
        <UPlotChart
          data={chartData[MetricType.Throughput]}
          options={chartOptions[MetricType.Throughput]}
          style={CHART_STYLE}
        />
      </Section>
      <Section
        bodyBorder
        bodyNoPadding
        title="Timing Metrics">
        <UPlotChart
          data={chartData[MetricType.Timing]}
          noDataMessage="No data found. Timing metrics may not be available for your framework."
          options={chartOptions[MetricType.Timing]}
          style={CHART_STYLE}
        />
      </Section>
      <Section
        bodyBorder
        bodyNoPadding
        filters={<SystemMetricFilter />}
        title="System Metrics">
        <UPlotChart
          data={chartData[MetricType.System]}
          options={chartOptions[MetricType.System]}
          style={CHART_STYLE}
        />
      </Section>
    </div>
  );
};

export default ProfilesEnabled;
