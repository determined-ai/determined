import React, { useMemo } from 'react';

import Pivot, { TabItem } from 'components/kit/Pivot';
import SplitPane from 'components/SplitPane';
import { ExperimentWithTrial, Project} from 'types';
import HpParallelCoordinates from './ExpParallelCoordinates';
import { MapOfIdsToColors } from './useGlasbey';

import CompareMetrics from './CompareMetrics';

interface Props {
  colorMap: MapOfIdsToColors;
  children: React.ReactElement;
  open: boolean;
  initialWidth: number;
  onWidthChange: (width: number) => void;
  selectedExperiments: ExperimentWithTrial[];
  project: Project
}
const ComparisonView: React.FC<Props> = ({
  children,
  open,
  initialWidth,
  onWidthChange,
  selectedExperiments,
}) => {
  const tabs: TabItem[] = useMemo(() => {
    return [
      {
        children: <CompareMetrics selectedExperiments={selectedExperiments} />,
        key: 'metrics',
        label: 'Metrics',
      },
      { key: 'hyperparameters', label: 'Hyperparameters' },
      { key: 'configurations', label: 'Configurations' },
    ];
  }, [selectedExperiments]);

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
