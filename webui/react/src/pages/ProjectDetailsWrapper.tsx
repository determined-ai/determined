import React from 'react';

import useFeature from 'hooks/useFeature';

import OldProjectDetails from './OldProjectDetails';
import ProjectDetails from './ProjectDetails';

const ProjectDetailsWrapper: React.FC = () => {
  // const trialsComparisonEnabled = useFeature().isOn('trials_comparison');
  // return trialsComparisonEnabled ? <ProjectDetails /> : <OldProjectDetails />;
  return <ProjectDetails />;
};

export default ProjectDetailsWrapper;
