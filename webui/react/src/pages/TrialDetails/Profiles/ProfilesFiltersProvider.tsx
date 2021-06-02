import { Alert } from 'antd';
import React, { createContext, useContext, useEffect, useMemo, useState } from 'react';

import { FiltersInterface } from 'pages/TrialDetails/Profiles/SystemMetricFilter';
import {
  AvailableSeriesType, MetricsAggregateInterface, MetricType, useFetchAvailableSeries,
  useFetchMetrics,
} from 'pages/TrialDetails/Profiles/utils';
import { TrialDetails } from 'types';

export interface ProfilesFiltersInterface {
  agentId?: string;
  gpuUuid?: string;
  name?: string;
}

export interface ProfilesFiltersContextInterface {
  filters: ProfilesFiltersInterface,
  hasProfilingData: boolean,
  setFilters: (value: ProfilesFiltersInterface) => void,
  setHasProfilingData: (value: boolean) => void,
  systemSeries: AvailableSeriesType,
  timingMetrics: MetricsAggregateInterface,
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
  const [ filters, setFilters ] = useState<FiltersInterface>({});
  const [ hasProfilingData, setHasProfilingData ] = useState<boolean>(false);
  const systemSeries = useFetchAvailableSeries(trial.id)[MetricType.System];
  const timingMetrics = useFetchMetrics(trial.id, MetricType.Timing);

  /*
   * Set default filters
   */
  useEffect(() => {
    if (!systemSeries || (filters.agentId && filters.name)) return;

    const newFilters: FiltersInterface = {
      agentId: filters.agentId,
      name: filters.name,
    };

    if (!filters.name) {
      if (Object.keys(systemSeries).includes('gpu_util')) newFilters.name = 'gpu_util';
      else if (Object.keys(systemSeries).includes('cpu_util')) newFilters.name = 'cpu_util';
      else newFilters.name = Object.keys(systemSeries)[0];
    }

    if (!filters.agentId) {
      newFilters.agentId = Object.keys(systemSeries[newFilters.name as unknown as string])[0];
    }

    setFilters(newFilters);
  }, [ systemSeries, filters.agentId, filters.name ]);

  const context = useMemo<ProfilesFiltersContextInterface>(() => ({
    filters,
    hasProfilingData,
    setFilters,
    setHasProfilingData,
    systemSeries,
    timingMetrics,
  }), [ filters, hasProfilingData, systemSeries, timingMetrics ]);

  if (!filters.agentId || !filters.name) {
    return <Alert message="No data available." type="warning" />;
  }

  return (
    <ProfilesFiltersContext.Provider value={context}>
      {children}
    </ProfilesFiltersContext.Provider>
  );
};

export default ProfilesFiltersProvider;
