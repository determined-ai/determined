import { Alert } from 'antd';
import React, { useEffect, useRef } from 'react';


import useScroll from 'hooks/useScroll';
import ProfilesEnabled from 'pages/TrialDetails/Profiles/ProfilesEnabled';
import ProfilesFiltersProvider from 'pages/TrialDetails/Profiles/ProfilesFiltersProvider';
import ProfilesNotEnabled from 'pages/TrialDetails/Profiles/ProfilesNotEnabled';
import { ExperimentBase, TrialDetails } from 'types';

import css from './TrialDetailsProfiles.module.scss';

export interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

export const CHART_HEIGHT = 400;

const Profiler: React.FC<Props> = ({ experiment, trial }: Props) => {
  if (!experiment.config.profiling?.enabled) {
    return (
      <ProfilesNotEnabled />
    );
  } else if (!trial) {
    return (
      <Alert
        message="Waiting for trial to become available."
        type="warning"
      />
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
  const containerRef = useRef<HTMLDivElement>(null);
  const scroll = useScroll(containerRef);
  const scrollTop = useRef(0);

  /*
   * Preserve and restore scroll position upon re-render.
   */

  useEffect(() => {
    if (containerRef.current && scroll.scrollTop === 0 && scrollTop.current !== 0) {
      containerRef.current.scrollTop = scrollTop.current;
    } else {
      scrollTop.current = scroll.scrollTop;
    }
  });
  return (
    <div className={css.base} ref={containerRef}>
      <Profiler {...props} />
    </div>
  );
};

export default TrialDetailsProfiles;
