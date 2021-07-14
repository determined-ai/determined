import { Alert } from 'antd';
import React, { createContext, useContext, useEffect, useMemo, useState } from 'react';

import { FiltersInterface } from 'pages/TrialDetails/Profiles/SystemMetricFilter';
import {
  AvailableSeriesType, MetricsAggregateInterface, MetricType, useFetchAvailableSeries,
  useFetchMetrics,
} from 'pages/TrialDetails/Profiles/utils';
import { parseUrl } from 'routes/utils';
import { TrialDetails } from 'types';

export interface ProfilesFiltersInterface {
  agentId?: string;
  gpuUuid?: string;
  name?: string;
}

export interface ProfilesFiltersContextInterface {
  filters: ProfilesFiltersInterface,
  setFilters: (value: ProfilesFiltersInterface) => void,
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
  const [ isUrlParsed, setIsUrlParsed ] = useState(false);
  const systemSeries = useFetchAvailableSeries(trial.id)[MetricType.System];
  const timingMetrics = useFetchMetrics(trial.id, MetricType.Timing);

  const canRender = filters.agentId && filters.name && systemSeries;

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

  /*
   * When filters changes update the page URL.
   */
  useEffect(() => {
    if (!isUrlParsed) return;

    const searchParams = new URLSearchParams;
    const url = parseUrl(window.location.href);

    // name
    if (filters.name) {
      searchParams.append('name', filters.name);
    }

    // agentId
    if (filters.agentId) {
      searchParams.append('agentId', filters.agentId);
    }

    // gpuUuid
    if (filters.gpuUuid) {
      searchParams.append('gpuUuid', filters.gpuUuid);
    }

    window.history.pushState(
      {},
      '',
      url.origin + url.pathname + '?' + searchParams.toString(),
    );
  }, [ filters, isUrlParsed ]);

  /*
   * On first load: if filters are specified in URL, override default.
   */
  useEffect(() => {
    if (!canRender || isUrlParsed) return;

    // If search params are not set, we default to user preferences
    const url = parseUrl(window.location.href);
    if (url.search === '') {
      setIsUrlParsed(true);
      return;
    }

    const newFilters = { ...filters };
    const urlSearchParams = url.searchParams;

    // name
    const name = urlSearchParams.get('name');
    if (name != null && Object.keys(systemSeries).includes(name)) {
      newFilters.name = name;
    }

    // agentId
    const agentId = urlSearchParams.get('agentId');
    if (
      agentId != null
      && newFilters.name
      && Object.keys(systemSeries[newFilters.name]).includes(agentId)
    ) {
      newFilters.agentId = agentId;
    }

    // gpuUuid
    const gpuUuid = urlSearchParams.get('gpuUuid');
    if (
      agentId != null
      && gpuUuid != null
      && newFilters.name
      && newFilters.agentId
      && systemSeries[newFilters.name][newFilters.agentId].includes(gpuUuid)
    ) {
      newFilters.gpuUuid = gpuUuid;
    }

    setIsUrlParsed(true);
    setFilters(newFilters);
  }, [ canRender, filters, isUrlParsed, systemSeries ]);

  const context = useMemo<ProfilesFiltersContextInterface>(() => ({
    filters,
    setFilters,
    systemSeries,
    timingMetrics,
  }), [ filters, systemSeries, timingMetrics ]);

  if (!canRender || !isUrlParsed) {
    return <Alert message="No data available." type="warning" />;
  }

  return (
    <ProfilesFiltersContext.Provider value={context}>
      {children}
    </ProfilesFiltersContext.Provider>
  );
};

export default ProfilesFiltersProvider;
