import React, { useEffect, useMemo, useState } from 'react';

import Pivot, { TabItem } from 'components/kit/Pivot';
import SplitPane from 'components/SplitPane';
import { isEqual } from 'shared/utils/data';
import { ExperimentWithTrial, TrialItem } from 'types';

import CompareMetrics from './CompareMetrics';
import CompareParallelCoordinates from './CompareParallelCoordinates';

interface Props {
  children: React.ReactElement;
  open: boolean;
  initialWidth: number;
  onWidthChange: (width: number) => void;
  projectId: number;
  selectedExperiments: ExperimentWithTrial[];
}
const ComparisonView: React.FC<Props> = ({
  children,
  open,
  initialWidth,
  onWidthChange,
  projectId,
  selectedExperiments,
}) => {
  const [trials, setTrials] = useState<TrialItem[]>([]);

  useEffect(() => {
    const ts: TrialItem[] = [];
    selectedExperiments.forEach((e) => e.bestTrial && ts.push(e.bestTrial));
    setTrials((prev: TrialItem[]) => {
      return isEqual(
        prev?.map((e) => e.id),
        ts?.map((e) => e?.id),
      )
        ? prev
        : ts;
    });
  }, [selectedExperiments]);

  const tabs: TabItem[] = useMemo(() => {
    return [
      {
        children: <CompareMetrics selectedExperiments={selectedExperiments} trials={trials} />,
        key: 'metrics',
        label: 'Metrics',
      },
      {
        children: (
          <CompareParallelCoordinates
            projectId={projectId}
            selectedExperiments={selectedExperiments}
            trials={trials}
          />
        ),
        key: 'hyperparameters',
        label: 'Hyperparameters',
      },
      { key: 'configurations', label: 'Configurations' },
    ];
  }, [selectedExperiments, projectId, trials]);

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
