import { string, undefined as undefinedType, union } from 'io-ts';
import React, { useEffect, useMemo } from 'react';

import { LineChart } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import Section from 'components/Section';
import { SettingsConfig, useSettings } from 'hooks/useSettings';
import SystemMetricFilter from 'pages/TrialDetails/Profiles/Charts/SystemMetricChartFilters';
import { ChartProps, MetricType } from 'pages/TrialDetails/Profiles/types';
import { useFetchProfilerMetrics } from 'pages/TrialDetails/Profiles/useFetchProfilerMetrics';
import { useFetchProfilerSeries } from 'pages/TrialDetails/Profiles/useFetchProfilerSeries';
import {
  getScientificNotationTickValues,
  getUnitForMetricName,
} from 'pages/TrialDetails/Profiles/utils';
import handleError from 'utils/error';

export interface Settings {
  agentId?: string;
  gpuUuid?: string;
  name?: string;
}

const config = (trialId: number): SettingsConfig<Settings> => ({
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
  storagePath: `profiler-filters-${trialId}`,
});

const SystemMetricChart: React.FC<ChartProps> = ({ trial }) => {
  const useSettingsConfig = useMemo(() => {
    return config(trial.id);
  }, [trial.id]);

  const { settings, updateSettings } = useSettings<Settings>(useSettingsConfig);

  const systemSeries = useFetchProfilerSeries(trial.id)[MetricType.System];

  const systemMetrics = useFetchProfilerMetrics(
    trial.id,
    trial.state,
    MetricType.System,
    settings.name,
    settings.agentId,
    settings.gpuUuid,
  );

  const yLabel = getUnitForMetricName(settings.name ?? '');

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
      <LineChart
        experimentId={trial.id}
        handleError={handleError}
        series={systemMetrics.data}
        xAxis={XAxisDomain.Time}
        xLabel="Time"
        yLabel={yLabel}
        yTickValues={getScientificNotationTickValues}
      />
    </Section>
  );
};

export default SystemMetricChart;
