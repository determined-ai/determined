import { string, undefined as undefinedType, union } from 'io-ts';
import React, { useEffect, useMemo } from 'react';

import Section from 'components/Section';
import UPlotChart from 'components/UPlot/UPlotChart';
import { SettingsConfig, useSettings } from 'hooks/useSettings';

import { ChartProps } from '../types';
import { MetricType } from '../types';
import { useFetchProfilerMetrics } from '../useFetchProfilerMetrics';
import { useFetchProfilerSeries } from '../useFetchProfilerSeries';

import SystemMetricFilter from './SystemMetricChartFilters';

export interface Settings {
  agentId?: string;
  gpuUuid?: string;
  name?: string;
}

const config: SettingsConfig<Settings> = {
  settings: {
    agentId: {
      defaultValue: undefined,
      storageKey: 'agentId',
      type: union([undefinedType, string]),
    },
    gpuUuid: {
      defaultValue: undefined,
      storageKey: 'gpuUuid',
      type: union([undefinedType, string]),
    },
    name: {
      defaultValue: undefined,
      storageKey: 'name',
      type: union([undefinedType, string]),
    },
  },
  storageKey: 'profiler-filters',
};

const SystemMetricChart: React.FC<ChartProps> = ({ getOptionsForMetrics, trial }) => {
  const { settings, updateSettings } = useSettings<Settings>(config);

  const systemSeries = useFetchProfilerSeries(trial.id)[MetricType.System];

  const systemMetrics = useFetchProfilerMetrics(
    trial.id,
    trial.state,
    MetricType.System,
    settings.name,
    settings.agentId,
    settings.gpuUuid,
  );

  const options = useMemo(
    () => getOptionsForMetrics(settings.name ?? '', systemMetrics.names),
    [getOptionsForMetrics, settings.name, systemMetrics.names],
  );

  useEffect(() => {
    if (!systemSeries || (settings.agentId && settings.name)) return;

    const newSettings: Partial<Settings> = {};

    if (!settings.name) {
      if (Object.keys(systemSeries).includes('gpu_util')) newSettings.name = 'gpu_util';
      else if (Object.keys(systemSeries).includes('cpu_util')) newSettings.name = 'cpu_util';
      else newSettings.name = Object.keys(systemSeries)[0];
    }

    if (!settings.agentId) {
      newSettings.agentId = Object.keys(systemSeries[newSettings.name as unknown as string])[0];
    }

    if (Object.keys(newSettings).length !== 0) updateSettings(newSettings);
  }, [settings.agentId, settings.name, systemSeries, updateSettings]);

  return (
    <Section
      bodyBorder
      bodyNoPadding
      filters={
        settings && (
          <SystemMetricFilter
            settings={settings}
            systemSeries={systemSeries}
            updateSettings={updateSettings}
          />
        )
      }
      title="System Metrics">
      <UPlotChart data={systemMetrics.data} options={options} />
    </Section>
  );
};

export default SystemMetricChart;
