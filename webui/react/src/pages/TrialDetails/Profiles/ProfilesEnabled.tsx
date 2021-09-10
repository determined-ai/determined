import { Alert } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useMemo, useRef } from 'react';
import uPlot, { AlignedData } from 'uplot';

import Section from 'components/Section';
import Spinner from 'components/Spinner';
import UPlotChart, { Options } from 'components/UPlotChart';
import useScroll from 'hooks/useScroll';
import { useProfilesFilterContext } from 'pages/TrialDetails/Profiles/ProfilesFiltersProvider';
import SystemMetricFilter from 'pages/TrialDetails/Profiles/SystemMetricFilter';
import { convertMetricsToUplotData, getUnitForMetricName } from 'pages/TrialDetails/Profiles/utils';
import { glasbeyColor } from 'utils/color';

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
  const scroll = useScroll(containerRef);
  const scrollTop = useRef(0);

  const isLoading = (
    metrics[MetricType.System].isLoading &&
    metrics[MetricType.Throughput].isLoading &&
    metrics[MetricType.Timing].isLoading
  );
  const isEmpty = (
    metrics[MetricType.System].isEmpty &&
    metrics[MetricType.Throughput].isEmpty &&
    metrics[MetricType.Timing].isEmpty
  );

  const chartInfo = useMemo(() => {
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
      [MetricType.System]: {
        axes: [ timeAxis, metricAxis(settings.name) ],
        data: convertMetricsToUplotData(
          metrics[MetricType.System].dataByTime,
          metrics[MetricType.System].names,
        ),
      },
      [MetricType.Throughput]: {
        axes: [ timeAxis, metricAxis('samples_per_second') ],
        data: convertMetricsToUplotData(
          metrics[MetricType.Throughput].dataByTime,
          metrics[MetricType.Throughput].names,
        ),
      },
      [MetricType.Timing]: {
        axes: [ timeAxis, { label: 'Seconds', scale: 'y' } ],
        data: convertMetricsToUplotData(
          metrics[MetricType.Timing].dataByTime,
          metrics[MetricType.Timing].names,
        ),
      },
    };

    // Finalize uPlot data and options for all charts.
    const metricKeys = Object.keys(metrics) as MetricType[];
    const uPlotData = metricKeys.reduce((acc, key) => {
      const [ times, batches ] = config[key].data;

      // Pad the series data with empty series data.
      const data = metricKeys.reduce((acc, seriesKey) => {
        const seriesData = config[seriesKey].data.slice(2);
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
      const series = metricKeys.reduce((acc, seriesKey) => {
        const metricNames = metrics[seriesKey].names.slice(1);
        if (seriesKey === key) {
          const series = metricNames.map(seriesMapping);
          acc.push(...series);
        } else {
          const filler = new Array(metricNames.length).fill(null).map(fillerMapping);
          acc.push(...filler);
        }
        return acc;
      }, [ timeSeries, batchSeries ]);

      const options = { ...sharedOptions, axes: config[key].axes, series } as Options;

      return { ...acc, [key]: { data, options } };
    }, {} as Record<MetricType, { data: AlignedData, options: Options }>);

    return uPlotData;
  }, [ metrics, settings.name ]);

  /*
   * Preserve and restore scroll position upon re-render.
   */
  useEffect(() => {
    if (containerRef.current && scroll.scrollTop === 0 && scrollTop.current !== 0) {
      containerRef.current.scrollTop = scrollTop.current;
    } else {
      scrollTop.current = scroll.scrollTop;
    }
  }, [ scroll ]);

  if (isLoading) {
    return <Spinner spinning tip="Fetching system metrics..." />;
  } else if (isEmpty) {
    return <Alert message="No data available." type="warning" />;
  }

  return (
    <div ref={containerRef}>
      <Section
        bodyBorder
        bodyNoPadding
        loading={metrics[MetricType.Throughput].isLoading}
        title="Throughput">
        <UPlotChart
          data={chartInfo[MetricType.Throughput].data}
          options={chartInfo[MetricType.Throughput].options}
          style={CHART_STYLE}
        />
      </Section>
      <Section
        bodyBorder={!metrics[MetricType.Timing].isEmpty}
        bodyNoPadding
        loading={metrics[MetricType.Timing].isLoading}
        title="Timing Metrics">
        {metrics[MetricType.Timing].isEmpty ? (
          <Alert
            description="Timing metrics may not be available for your framework."
            message="No data found."
            type="warning"
          />
        ) : (
          <UPlotChart
            data={chartInfo[MetricType.Timing].data}
            options={chartInfo[MetricType.Timing].options}
            style={CHART_STYLE}
          />
        )}
      </Section>
      <Section
        bodyBorder
        bodyNoPadding
        filters={<SystemMetricFilter />}
        loading={metrics[MetricType.System].isLoading}
        title="System Metrics">
        <UPlotChart
          data={chartInfo[MetricType.System].data}
          options={chartInfo[MetricType.System].options}
          style={CHART_STYLE}
        />
      </Section>
    </div>
  );
};

export default ProfilesEnabled;
