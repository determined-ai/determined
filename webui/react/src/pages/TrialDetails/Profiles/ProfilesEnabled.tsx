import { Alert } from 'antd';
import React from 'react';

import Section from 'components/Section';
import Spinner from 'components/Spinner';
import { useProfilesFilterContext } from 'pages/TrialDetails/Profiles/ProfilesFiltersProvider';
import SystemMetricChart from 'pages/TrialDetails/Profiles/SystemMetricChart';
import SystemMetricFilter from 'pages/TrialDetails/Profiles/SystemMetricFilter';
import ThroughputMetricChart from 'pages/TrialDetails/Profiles/ThroughputMetricChart';
import TimingMetricChart from 'pages/TrialDetails/Profiles/TimingMetricChart';
import { MetricType, useFetchMetrics } from 'pages/TrialDetails/Profiles/utils';
import { TrialDetails } from 'types';

export interface Props {
  trial: TrialDetails;
}

const ProfilesEnabled: React.FC<Props> = ({ trial }: Props) => {
  const {
    filters,
    timingMetrics,
  } = useProfilesFilterContext();

  const systemMetrics = useFetchMetrics(
    trial.id,
    MetricType.System,
    filters.name,
    filters.agentId,
    filters.gpuUuid,
  );

  const throughtputMetrics = useFetchMetrics(
    trial.id,
    MetricType.Throughput,
    'samples_per_second',
    undefined,
    undefined,
  );

  const isLoading = systemMetrics.isLoading || throughtputMetrics.isLoading;
  const isEmpty = systemMetrics.isEmpty && throughtputMetrics.isEmpty;

  if (isLoading) {
    return <Spinner spinning={isLoading} tip="Fetching system metrics..." />;
  } else if (isEmpty) {
    return <Alert message="No data available." type="warning" />;
  }

  return (
    <>
      <Section
        bodyBorder
        loading={throughtputMetrics.isLoading}
        title="Throughput">
        <ThroughputMetricChart throughputMetrics={throughtputMetrics} />
      </Section>
      <Section
        bodyBorder={!timingMetrics.isEmpty}
        loading={timingMetrics.isLoading}
        title="Timing Metrics">
        {timingMetrics.isEmpty
          ? <Alert
            description="Timing metrics may not be available for your framework."
            message="No data found."
            type="warning"
          />
          : <TimingMetricChart timingMetrics={timingMetrics} />
        }
      </Section>
      <Section
        bodyBorder
        filters={<SystemMetricFilter />}
        loading={systemMetrics.isLoading}
        title="System Metrics">
        <SystemMetricChart systemMetrics={systemMetrics} />
      </Section>
    </>
  );
};

export default ProfilesEnabled;
