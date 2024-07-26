import Alert from 'hew/Alert';
import React from 'react';

import useFeature from 'hooks/useFeature';
import Profiler from 'pages/TrialDetails/Profiles/Profiler';
import { TrialDetails } from 'types';

import css from './TrialDetailsProfiles.module.scss';

export interface Props {
  trial?: TrialDetails;
}

export const CHART_HEIGHT = 400;

const TrialDetailsProfiles: React.FC<Props> = ({ trial }: Props) => {
  const f_flat_runs = useFeature().isOn('flat_runs');
  return (
    <div className={css.base}>
      {!trial ? (
        <Alert
          message={`Waiting for ${f_flat_runs ? 'run' : 'trial'} to become available.`}
          type="warning"
        />
      ) : (
        <Profiler trial={trial} />
      )}
    </div>
  );
};

export default TrialDetailsProfiles;
