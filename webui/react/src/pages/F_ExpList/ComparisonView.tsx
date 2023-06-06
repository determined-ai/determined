import React, { useMemo } from 'react';

import Pivot, { TabItem } from 'components/kit/Pivot';
import SplitPane from 'components/SplitPane';
import { ExperimentWithTrial } from 'types';

import CompareMetrics from './CompareMetrics';
import CompareParallelCoordinates from './CompareParallelCoordinates';

interface Props {
  children: React.ReactElement;
  open: boolean;
  initialWidth: number;
  onWidthChange: (width: number) => void;
  selectedExperiments: ExperimentWithTrial[];
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
      {
        children: <CompareParallelCoordinates selectedExperiments={selectedExperiments} />,
        key: 'hyperparameters',
        label: 'Hyperparameters',
      },
      { key: 'configurations', label: 'Configurations' },
    ];
  }, [selectedExperiments]);

  return (
    <div>
      <SplitPane initialWidth={initialWidth} open={open} onChange={onWidthChange}>
        {children}
        <Pivot destroyInactiveTabPane items={tabs} />
      </SplitPane>
    </div>
  );
};

export default ComparisonView;
