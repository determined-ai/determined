import React, { useMemo } from 'react';

import Pivot, { TabItem } from 'components/kit/Pivot';
import SplitPane from 'components/SplitPane';
import { ExperimentWithTrial, Project} from 'types';
import HpParallelCoordinates from './ExpParallelCoordinates';
import { MapOfIdsToColors } from './useGlasbey';
interface Props {
  colorMap: MapOfIdsToColors;
  children: React.ReactElement;
  open: boolean;
  initialWidth: number;
  onWidthChange: (width: number) => void;
  selectedExperiments: ExperimentWithTrial[];
  project: Project
}

const ComparisonView: React.FC<Props> = ({ children, open, initialWidth, onWidthChange,  selectedExperiments, project, colorMap}) => {

  const tabs: TabItem[] = useMemo(() => {
    return [
      { key: 'metrics', label: 'Metrics'},
      { key: 'hyperparameters', label: 'Hyperparameters', children:<HpParallelCoordinates colorMap={colorMap} workspaceId={project.workspaceId} experiments={selectedExperiments}/>  },
      { key: 'configurations', label: 'Configurations' },
    ];
  }, [selectedExperiments, project]);

  return (
    <div>
      <SplitPane initialWidth={initialWidth} open={open} onChange={onWidthChange}>
        {children}
        <Pivot items={tabs} />
      </SplitPane>
    </div>
  );
};

export default ComparisonView;
