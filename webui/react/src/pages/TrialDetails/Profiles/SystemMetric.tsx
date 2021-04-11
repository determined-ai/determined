import React, { useState } from 'react';

import Section from 'components/Section';
import { TrialDetails } from 'types';

import SystemMetricChart from './SystemMetricChart';
import SystemMetricFilter, { FiltersInterface } from './SystemMetricFilter';

export interface Props {
  trial: TrialDetails;
}

const SystemMetric: React.FC<Props> = ({ trial }: Props) => {
  const [ filters, setFilters ] = useState<FiltersInterface>({});

  return (
    <Section
      bodyBorder
      filters={<SystemMetricFilter trial={trial} value={filters} onChange={setFilters} />}
      title="System Metrics"
    >
      <SystemMetricChart filters={filters} trial={trial} />
    </Section>
  );
};

export default SystemMetric;
