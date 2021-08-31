import { Alert } from 'antd';
import dayjs from 'dayjs';
import React, { useMemo, useRef } from 'react';
import uPlot from 'uplot';

import Section from 'components/Section';
import Spinner from 'components/Spinner';
import UPlotChart, { Options } from 'components/UPlotChart';
import { useProfilesFilterContext } from 'pages/TrialDetails/Profiles/ProfilesFiltersProvider';
import SystemMetricFilter from 'pages/TrialDetails/Profiles/SystemMetricFilter';
import {
  convertMetricsToUplotData, getUnitForMetricName, MetricType, useFetchMetrics,
} from 'pages/TrialDetails/Profiles/utils';
import { TrialDetails } from 'types';
import { glasbeyColor } from 'utils/color';
import { findFactorOfNumber } from 'utils/number';

export interface Props {
  trial: TrialDetails;
}

const CHART_HEIGHT = 300;

const chartStyle: React.CSSProperties = { paddingBottom: 16 };

const ProfilesEnabled: React.FC<Props> = ({ trial }: Props) => {
  const chartSyncKey = useRef(uPlot.sync('time'));

  const { filters, timingMetrics } = useProfilesFilterContext();

  const systemMetrics = useFetchMetrics(
    trial.id,
    MetricType.System,
    filters.name,
    filters.agentId,
    filters.gpuUuid,
  );

  const throughputMetrics = useFetchMetrics(
    trial.id,
    MetricType.Throughput,
    'samples_per_second',
    undefined,
    undefined,
  );

  const isLoading = systemMetrics.isLoading || throughputMetrics.isLoading;
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

    return {
      [MetricType.System]: {
        data: convertMetricsToUplotData(systemMetrics.dataByUnixTime, systemMetrics.names),
        options: {
          ...sharedOptions,
          axes: [
            {
              label: 'Time',
              space: (self, axisIdx, scaleMin, scaleMax, plotDim) => {
                const rangeMs = scaleMax - scaleMin;
                const msPerSec = plotDim / rangeMs;
                return Math.max(60, msPerSec * 10 * 1000);
              },
              values: (self, splits) => {
                return splits.map(i => dayjs.utc(i).format('HH:mm:ss'));
              },
            },
            {
              label: getUnitForMetricName(filters.name || ''),
              size: (self: uPlot, values: string[]) => {
                if (!values) return 50;
                const maxChars = Math.max(...values.map(el => el.toString().length));
                return 25 + Math.max(25, maxChars * 8);
              },
            },
          ],
          series: [
            {
              label: 'Time',
              value: (self, rawValue) => {
                return dayjs.utc(rawValue).format('HH:mm:ss.SSS');
              },
            },
            ...systemMetrics.names.map((name, index) => ({
              label: name,
              points: { show: false },
              stroke: glasbeyColor(index),
              width: 2,
            })),
          ],
        } as Options,
      },
      [MetricType.Throughput]: {
        data: convertMetricsToUplotData(throughputMetrics.dataByUnixTime, throughputMetrics.names),
        options: {
          ...sharedOptions,
          axes: [
            {
              label: 'Time',
              space: (self, axisIdx, scaleMin, scaleMax, plotDim) => {
                const rangeMs = scaleMax - scaleMin;
                const msPerSec = plotDim / rangeMs;
                return Math.max(60, msPerSec * 10 * 1000);
              },
              values: (self, splits) => {
                return splits.map(i => dayjs.utc(i).format('HH:mm:ss'));
              },
            },
            {
              label: getUnitForMetricName('samples_per_second'),
              size: (self: uPlot, values: string[]) => {
                if (!values) return 50;
                const maxChars = Math.max(...values.map(el => el.toString().length));
                return 25 + Math.max(25, maxChars * 8);
              },
            },
          ],
          series: [
            {
              label: 'Time',
              value: (self, rawValue) => {
                return dayjs.utc(rawValue).format('HH:mm:ss.SSS');
              },
            },
            ...throughputMetrics.names.map((name, index) => ({
              label: name,
              points: { show: false },
              stroke: glasbeyColor(index),
              width: 2,
            })),
          ],
        } as Options,
      },
      [MetricType.Timing]: {
        data: convertMetricsToUplotData(timingMetrics.dataByBatch, timingMetrics.names),
        options: {
          ...sharedOptions,
          axes: [
            {
              label: 'Batch',
              space: (self, axisIdx, scaleMin, scaleMax, plotDim) => {
                const range = scaleMax - scaleMin + 1;
                const factor = findFactorOfNumber(range).reverse()
                  .find(factor => plotDim / factor > 35);
                return factor ? Math.min(70, (plotDim / factor)) : 35;
              },
            },
            { label: 'Seconds' },
          ],
          series: [
            { label: 'Batch' },
            ...timingMetrics.names.map((name, index) => ({
              label: name,
              points: { show: false },
              stroke: glasbeyColor(index),
              width: 2,
            })),
          ],
        } as Options,
      },
    };
  }, [ filters.name, systemMetrics, throughputMetrics, timingMetrics ]);

  if (isLoading) {
    return <Spinner spinning={isLoading} tip="Fetching system metrics..." />;
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
