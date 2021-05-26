import { Alert } from 'antd';
import React from 'react';

import Link from 'components/Link';
import { paths } from 'routes/utils';
import { ExperimentBase, TrialDetails } from 'types';

import ProfilesEnabled from './Profiles/ProfilesEnabled';

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

  return <ProfilesEnabled trial={trial} />;
};

export default TrialDetailsProfiles;
