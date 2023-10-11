import React, { useMemo } from 'react';

import Message from 'components/kit/Message';
import Pivot, { TabItem } from 'components/kit/Pivot';
import SplitPane from 'components/SplitPane';
import useScrollbarWidth from 'hooks/useScrollbarWidth';
import { TrialsComparisonTable } from 'pages/ExperimentDetails/TrialsComparisonModal';
import { useTrialMetrics } from 'pages/TrialDetails/useTrialMetrics';
import { ExperimentWithTrial, TrialItem } from 'types';

import CompareMetrics from './CompareMetrics';
import CompareParallelCoordinates from './CompareParallelCoordinates';
import { MIN_COLUMN_WIDTH } from './glide-table/columns';

interface Props {
  children: React.ReactElement;
  open: boolean;
  initialWidth: number;
  onWidthChange: (width: number) => void;
  fixedColumnsCount: number;
  projectId: number;
  selectedExperiments: ExperimentWithTrial[];
}
const ComparisonView: React.FC<Props> = ({
  children,
  open,
  initialWidth,
  onWidthChange,
  fixedColumnsCount,
  projectId,
  selectedExperiments,
}) => {
  const scrollbarWidth = useScrollbarWidth();
  const hasPinnedColumns = fixedColumnsCount > 1;

  const minWidths: [number, number] = useMemo(() => {
    return [fixedColumnsCount * MIN_COLUMN_WIDTH + scrollbarWidth, 100];
  }, [fixedColumnsCount, scrollbarWidth]);

  const trials = useMemo(
    () =>
      selectedExperiments.filter((exp) => !!exp.bestTrial).map((exp) => exp.bestTrial as TrialItem),
    [selectedExperiments],
  );

  const experiments = useMemo(
    () => selectedExperiments.map((exp) => exp.experiment),
    [selectedExperiments],
  );

  const metricData = useTrialMetrics(trials);

  const tabs: TabItem[] = useMemo(() => {
    return [
      {
        children: (
          <CompareMetrics
            metricData={metricData}
            selectedExperiments={selectedExperiments}
            trials={trials}
          />
        ),
        key: 'metrics',
        label: 'Metrics',
      },
      {
        children: (
          <CompareParallelCoordinates
            metricData={metricData}
            projectId={projectId}
            selectedExperiments={selectedExperiments}
            trials={trials}
          />
        ),
        key: 'hyperparameters',
        label: 'Hyperparameters',
      },
      {
        children: <TrialsComparisonTable experiment={experiments} trials={trials} />,
        key: 'details',
        label: 'Details',
      },
    ];
  }, [selectedExperiments, projectId, experiments, trials, metricData]);

  return (
    <SplitPane
      initialWidth={initialWidth}
      minimumWidths={minWidths}
      open={open}
      onChange={onWidthChange}>
      {open && !hasPinnedColumns ? (
        <Message icon="info" title='Pin columns to see them in "Compare View"' />
      ) : (
        children
      )}
      {selectedExperiments.length === 0 ? (
        <Message
          description="Select experiments you would like to compare."
          icon="warning"
          title="No experiments selected."
        />
      ) : (
        <Pivot items={tabs} />
      )}
    </SplitPane>
  );
};

export default ComparisonView;
