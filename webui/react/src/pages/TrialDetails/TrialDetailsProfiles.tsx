import React from 'react';

import ProfilesEnabled from 'pages/TrialDetails/Profiles/ProfilesEnabled';
import ProfilesFiltersProvider from 'pages/TrialDetails/Profiles/ProfilesFiltersProvider';
import ProfilesNotEnabled from 'pages/TrialDetails/Profiles/ProfilesNotEnabled';
import { ExperimentBase, TrialDetails } from 'types';

import css from './TrialDetailsProfiles.module.scss';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

export const CHART_HEIGHT = 400;

const Profiler: React.FC<Props> = ({ experiment, trial }: Props) => {
  if (!experiment.config.profiling?.enabled) {
    return (
      <ProfilesNotEnabled />
    );
  } else {
    return (
      <ProfilesFiltersProvider trial={trial}>
        <ProfilesEnabled />
      </ProfilesFiltersProvider>
    );
  }
};

const TrialDetailsProfiles: React.FC<Props> = (props: Props) => {
  return (
    <div className={css.base}>
      <Profiler {...props} />
    </div>
  );
};

export default TrialDetailsProfiles;
