import { Alert } from 'antd';
import React from 'react';

import Link from 'components/Link';
import Profiler from 'pages/TrialDetails/Profiles/Profiler';
import { paths } from 'routes/utils';
import { ExperimentBase, TrialDetails } from 'types';

import css from './TrialDetailsProfiles.module.scss';

export interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

export const CHART_HEIGHT = 400;

const TrialDetailsProfiles: React.FC<Props> = ({ experiment, trial }: Props) => {
  return (
    <div className={css.base}>
      {!experiment.config.profiling?.enabled ? (
        <Alert
          description={
            <>
              Learn about&nbsp;
              <Link
                external
                path={paths.docs('/training-apis/experiment-config.html#profiling')}
                popout>
                how to enable profiling on trials
              </Link>
              .
            </>
          }
          message="Profiling was not enabled for this trial."
          type="warning"
        />
      ) : !trial ? (
        <Alert message="Waiting for trial to become available." type="warning" />
      ) : (
        <Profiler trial={trial} />
      )}
    </div>
  );
};

export default TrialDetailsProfiles;
