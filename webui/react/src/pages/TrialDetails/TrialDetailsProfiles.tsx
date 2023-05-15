import { Alert } from 'antd';
import React from 'react';

import Empty from 'components/kit/Empty';
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
        <Empty
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
        <Alert message="Waiting for trial to become available." type="warning" />
      ) : (
        <Profiler trial={trial} />
      )}
    </div>
  );
};

export default TrialDetailsProfiles;
