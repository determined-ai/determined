import Alert from 'hew/Alert';
import React from 'react';

import Profiler from 'pages/TrialDetails/Profiles/Profiler';
import { TrialDetails } from 'types';

import css from './TrialDetailsProfiles.module.scss';

export interface Props {
  trial?: TrialDetails;
}

export const CHART_HEIGHT = 400;

const TrialDetailsProfiles: React.FC<Props> = ({ trial }: Props) => {
  return (
    <div className={css.base}>
      {!trial ? (
        <Alert message="Waiting for trial to become available." type="warning" />
      ) : (
        <Profiler trial={trial} />
      )}
    </div>
  );
};

export default TrialDetailsProfiles;
