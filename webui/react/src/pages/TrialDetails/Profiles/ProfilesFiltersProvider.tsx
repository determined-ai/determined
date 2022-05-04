import React, { createContext, useContext, useEffect, useMemo, useState } from 'react';

import usePolling from 'hooks/usePolling';
import useSettings from 'hooks/useSettings';
import { TrialDetails } from 'types';

import settingsConfig, { Settings } from './ProfilesFiltersProvider.settings';
import { AvailableSeriesType, MetricsAggregateInterface, MetricType } from './types';
import { useFetchAvailableSeries } from './useFetchAvailableSeries';
import { useFetchMetrics } from './useFetchMetrics';

export interface ProfilesFiltersInterface {
  agentId?: string;
  gpuUuid?: string;
  name?: string;
}

export interface ProfilesFiltersContextInterface {
  metrics: Record<MetricType, MetricsAggregateInterface>,
  settings: Settings,
  systemSeries: AvailableSeriesType,
  updateSettings: (newSettings: Partial<Settings>, push?: boolean) => void;
}

export const ProfilesFiltersContext =
  createContext<ProfilesFiltersContextInterface|undefined>(undefined);

export const useProfilesFilterContext = (): ProfilesFiltersContextInterface => {
  const context = useContext(ProfilesFiltersContext);
  if (context === undefined) {
    throw new Error('useProfilesFilterContext must be used within a ProfilesFiltersContext');
  }
  return context;
};

interface Props {
  children: React.ReactNode;
  trial: TrialDetails;
}

const ProfilesFiltersProvider: React.FC<Props> = ({ children, trial }: Props) => {
  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);
  const systemSeries = useFetchAvailableSeries(trial.id)[MetricType.System];
  const systemMetrics = useFetchMetrics(
    trial.id,
    trial.state,
    MetricType.System,
    settings.name,
    settings.agentId,
    settings.gpuUuid,
  );
  const throughputMetrics = useFetchMetrics(
    trial.id,
    trial.state,
    MetricType.Throughput,
    'samples_per_second',
    undefined,
    undefined,
  );
  const timingMetrics = useFetchMetrics(
    trial.id,
    trial.state,
    MetricType.Timing,
  );

  /*
   * Set default filter settings.
   */
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
  }, [ settings.agentId, settings.name, systemSeries, updateSettings ]);

  const [ pollSignal, setPollSignal ] = useState(false);
  usePolling(() => {
    setPollSignal(prevPollSignal => !prevPollSignal);
  }, { interval: 2000 });

  const context = useMemo<ProfilesFiltersContextInterface>(
    () =>
      ({
        metrics: {
          [MetricType.System]: systemMetrics,
          [MetricType.Throughput]: throughputMetrics,
          [MetricType.Timing]: timingMetrics,
        },
        settings,
        systemSeries,
        updateSettings,
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps
    , [ pollSignal,
      settings.name,
      settings.agentId,
      settings.gpuUuid,
      systemMetrics.isLoading,
      throughputMetrics.isLoading,
      timingMetrics.isLoading,
      systemMetrics.isEmpty,
      throughputMetrics.isEmpty,
      timingMetrics.isEmpty,
    ],
  );

  return (
    <ProfilesFiltersContext.Provider value={context}>
      {children}
    </ProfilesFiltersContext.Provider>
  );
};

export default ProfilesFiltersProvider;
