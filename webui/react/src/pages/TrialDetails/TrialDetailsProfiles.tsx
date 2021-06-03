import { Alert } from 'antd';
import React from 'react';

import Link from 'components/Link';
import ProfilesEnabled from 'pages/TrialDetails/Profiles/ProfilesEnabled';
import ProfilesFiltersProvider from 'pages/TrialDetails/Profiles/ProfilesFiltersProvider';
import { paths } from 'routes/utils';
import { ExperimentBase, TrialDetails } from 'types';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

export const CHART_HEIGHT = 400;

const TrialDetailsProfiles: React.FC<Props> = ({ experiment, trial }: Props) => {
  if (!experiment.config.profiling?.enabled) {
    const description = (
      <>
        Learn about ;
        <Link
          external
          path={paths.docs('/reference/experiment-config.html#searcher')} // todo: change me
          popout>how to enable profiling on trials</Link>.
      </>
    );

    return (
      <Alert
        description={description}
        message="Profiling was not enabled for this trial."
        type="warning"
      />
    );
  }

  return (
    <ProfilesFiltersProvider trial={trial}>
      <ProfilesEnabled trial={trial} />
    </ProfilesFiltersProvider>
  );
};

export default TrialDetailsProfiles;
