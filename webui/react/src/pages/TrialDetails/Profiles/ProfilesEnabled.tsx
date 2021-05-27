import { Alert } from 'antd';
import React, { useState } from 'react';

import Section from 'components/Section';
import { TrialDetails } from 'types';

import SystemMetricChart from './SystemMetricChart';
import SystemMetricFilter, { FiltersInterface } from './SystemMetricFilter';
import TimingMetricChart from './TimingMetricChart';
import { MetricType, useFetchMetrics } from './utils';

export interface Props {
  trial: TrialDetails;
}

const ProfilesEnabled: React.FC<Props> = ({ trial }: Props) => {
  const [ filters, setFilters ] = useState<FiltersInterface>({});
  const systemMetrics = useFetchMetrics(
    trial.id,
    MetricType.System,
    filters.name,
    filters.agentId,
    filters.gpuUuid,
  );

  return (
    <>

      {systemMetrics.isEmpty && (
        <Alert
          message="No data available."
          type="warning"
        />
      )}

      <div style={{ display: (systemMetrics.isEmpty ? 'none' : 'block') }}>

        <Section
          bodyBorder
          filters={<SystemMetricFilter trial={trial} value={filters} onChange={setFilters} />}
          title="System Metrics"
        >
          <SystemMetricChart systemMetrics={systemMetrics} />
        </Section>

        <TimingMetricChart trial={trial} />

      </div>

    </>
  );
};

export default ProfilesEnabled;
