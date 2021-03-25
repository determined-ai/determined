import React, { useState } from 'react';

import Section from 'components/Section';
import { ExperimentBase, TrialDetails } from 'types';

import ProfilesNotEnabled from './Profiles/ProfilesNotEnabled';
import SystemMetricChart from './Profiles/SystemMetricChart';
import SystemMetricFilter, { FiltersInterface } from './Profiles/SystemMetricFilter';
import TimingMetricChart from './Profiles/TimingMetricChart';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

export const CHART_HEIGHT = 400;

const TrialDetailsProfiles: React.FC<Props> = ({ experiment, trial }: Props) => {
  const [ filters, setFilters ] = useState<FiltersInterface>({});
  const isProfilingEnabled = experiment.config.profiling?.enabled;

  if (!isProfilingEnabled) {
    return <ProfilesNotEnabled />;
  }

  return (
    <>

      <Section
        bodyBorder
        filters={<SystemMetricFilter trial={trial} value={filters} onChange={setFilters} />}
        title="System Metrics"
      >
        <SystemMetricChart filters={filters} trial={trial} />
      </Section>

      <Section bodyBorder title="Timing Metrics">
        <TimingMetricChart trial={trial} />
      </Section>

    </>
  );
};

export default TrialDetailsProfiles;
