import React from 'react';

import Message from 'components/kit/Message';
import Link from 'components/Link';
import Profiler from 'pages/TrialDetails/Profiles/Profiler';
import { paths } from 'routes/utils';
import { ExperimentBase, TrialDetails } from 'types';

import css from './TrialDetailsProfiles.module.scss';

export interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

const TrialDetailsProfiles: React.FC<Props> = ({ experiment, trial }: Props) => {
  return (
    <div className={css.base}>
      {!experiment.config.profiling?.enabled ? (
        <Message
          description={
            <>
              Enable experiment profiling to analyze and debug model system performance.&nbsp;
              <Link
                external
                path={paths.docs('/reference/training/experiment-config-reference.html#profiling')}>
                Get started
              </Link>
            </>
          }
          title="No profiling enabled"
        />
      ) : !trial ? (
        <Message icon="warning" title="Waiting for trial to become available." />
      ) : (
        <Profiler trial={trial} />
      )}
    </div>
  );
};

export default TrialDetailsProfiles;
