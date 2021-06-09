import React from 'react';

import ProfilesEnabled from 'pages/TrialDetails/Profiles/ProfilesEnabled';
import ProfilesFiltersProvider from 'pages/TrialDetails/Profiles/ProfilesFiltersProvider';
import ProfilesNotEnabled from 'pages/TrialDetails/Profiles/ProfilesNotEnabled';
import { ExperimentBase, TrialDetails } from 'types';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

export const CHART_HEIGHT = 400;

const TrialDetailsProfiles: React.FC<Props> = ({ experiment, trial }: Props) => {
  if (!experiment.config.profiling?.enabled) {
    return (
      <ProfilesNotEnabled />
    );
  } else {
    return (
      <ProfilesFiltersProvider trial={trial}>
        <ProfilesEnabled trial={trial} />
      </ProfilesFiltersProvider>
    );
  }
};

export default TrialDetailsProfiles;
