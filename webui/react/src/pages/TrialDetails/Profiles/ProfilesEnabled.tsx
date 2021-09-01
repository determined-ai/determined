import { Alert } from 'antd';
import dayjs from 'dayjs';
import React, { useMemo, useRef } from 'react';
import uPlot from 'uplot';

import Section from 'components/Section';
import Spinner from 'components/Spinner';
import UPlotChart, { Options } from 'components/UPlotChart';
import { useProfilesFilterContext } from 'pages/TrialDetails/Profiles/ProfilesFiltersProvider';
import SystemMetricFilter from 'pages/TrialDetails/Profiles/SystemMetricFilter';
import { convertMetricsToUplotData, getUnitForMetricName } from 'pages/TrialDetails/Profiles/utils';
import { glasbeyColor } from 'utils/color';

import css from './ProfilesEnabled.module.scss';
import { MetricType } from './types';

const CHART_HEIGHT = 300;

const chartStyle: React.CSSProperties = { paddingBottom: 16 };

const ProfilesEnabled: React.FC = () => {
  const chartSyncKey = useRef(uPlot.sync('time'));

  const { filters, throughputMetrics, timingMetrics, systemMetrics } = useProfilesFilterContext();

  const isLoading = (
    systemMetrics.isLoading && throughputMetrics.isLoading && timingMetrics.isLoading
  );
  const isEmpty = systemMetrics.isEmpty && throughputMetrics.isEmpty;

  const chartInfo = useMemo(() => {
    const tzDate = (ts: number) => uPlot.tzDate(new Date(ts * 1e3), 'Etc/UTC');
    const matchSyncKeys: uPlot.Cursor.Sync.ScaleKeyMatcher = (own, ext) => {
      return own === ext;
    };

    const cursorOptions: uPlot.Cursor = {
      focus: { prox: 16 },
      lock: true,
      sync: {
        key: chartSyncKey.current.key,
        match: [ matchSyncKeys, matchSyncKeys ],
        setSeries: true,
      },
    };

    const sharedOptions = {
      cursor: cursorOptions,
      height: CHART_HEIGHT,
      scales: { x: { time: false } },
      tzDate,
    };

    const timeAxis: uPlot.Axis = {
      label: 'Time',
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
      scale: 'metric',
      size: (self: uPlot, values: string[]) => {
        if (!values) return 50;
        const maxChars = Math.max(...values.map(el => el.toString().length));
        return 25 + Math.max(25, maxChars * 8);
      },
    });
    const timeSeries: uPlot.Series = {
      label: 'Time',
      value: (self, rawValue) => {
        return dayjs.utc(rawValue).format('HH:mm:ss.SSS');
      },
    };
    const batchSeries: uPlot.Series = { class: css.batchLegend, label: 'Batch', show: false };

    const seriesMapping = (name: string, index: number) => ({
      label: name,
      points: { show: false },
      scale: 'metric',
      spanGaps: true,
      stroke: glasbeyColor(index),
      width: 2,
    });

    return {
      [MetricType.System]: {
        data: convertMetricsToUplotData(systemMetrics.dataByTime, systemMetrics.names),
        options: {
          ...sharedOptions,
          axes: [ timeAxis, metricAxis(filters.name) ],
          series: [
            timeSeries,
            batchSeries,
            ...(systemMetrics.names.slice(1)).map(seriesMapping),
          ],
        } as Options,
      },
      [MetricType.Throughput]: {
        data: convertMetricsToUplotData(throughputMetrics.dataByTime, throughputMetrics.names),
        options: {
          ...sharedOptions,
          axes: [ timeAxis, metricAxis('samples_per_second') ],
          series: [
            timeSeries,
            batchSeries,
            ...(throughputMetrics.names.slice(1)).map(seriesMapping),
          ],
        } as Options,
      },
      [MetricType.Timing]: {
        data: convertMetricsToUplotData(timingMetrics.dataByTime, timingMetrics.names),
        options: {
          ...sharedOptions,
          axes: [
            timeAxis,
            { label: 'Seconds', scale: 'metric' },
          ],
          series: [
            timeSeries,
            batchSeries,
            ...(timingMetrics.names.slice(1)).map(seriesMapping),
          ],
        } as Options,
      },
    };
  }, [ filters.name, systemMetrics, throughputMetrics, timingMetrics ]);

  if (isLoading) {
    return <Spinner spinning tip="Fetching system metrics..." />;
  } else if (isEmpty) {
    return <Alert message="No data available." type="warning" />;
  }

  return (
    <>
      <Section
        bodyBorder
        bodyNoPadding
        loading={throughputMetrics.isLoading}
        title="Throughput">
        <UPlotChart
          data={chartInfo[MetricType.Throughput].data}
          options={chartInfo[MetricType.Throughput].options}
          style={chartStyle}
        />
      </Section>
      <Section
        bodyBorder={!timingMetrics.isEmpty}
        bodyNoPadding
        loading={timingMetrics.isLoading}
        title="Timing Metrics">
        {timingMetrics.isEmpty ? (
          <Alert
            description="Timing metrics may not be available for your framework."
            message="No data found."
            type="warning"
          />
        ) : (
          <UPlotChart
            data={chartInfo[MetricType.Timing].data}
            options={chartInfo[MetricType.Timing].options}
            style={chartStyle}
          />
        )}
      </Section>
      <Section
        bodyBorder
        bodyNoPadding
        filters={<SystemMetricFilter />}
        loading={systemMetrics.isLoading}
        title="System Metrics">
        <UPlotChart
          data={chartInfo[MetricType.System].data}
          options={chartInfo[MetricType.System].options}
          style={chartStyle}
        />
      </Section>
    </>
  );
};

export default ProfilesEnabled;
