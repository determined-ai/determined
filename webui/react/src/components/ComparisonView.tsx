import Alert from 'hew/Alert';
import { MIN_COLUMN_WIDTH } from 'hew/DataGrid/columns';
import Message from 'hew/Message';
import Pivot, { PivotProps } from 'hew/Pivot';
import SplitPane, { Pane } from 'hew/SplitPane';
import React, { useMemo } from 'react';

import CompareHyperparameters from 'components/CompareHyperparameters';
import { MapOfIdsToColors } from 'hooks/useGlasbey';
import { useMetrics } from 'hooks/useMetrics';
import useMobile from 'hooks/useMobile';
import useScrollbarWidth from 'hooks/useScrollbarWidth';
import { TrialsComparisonTable } from 'pages/ExperimentDetails/TrialsComparisonModal';
import { ExperimentWithTrial, FlatRun, XOR } from 'types';

import CompareMetrics from './CompareMetrics';

export const EMPTY_MESSAGE = 'No items selected.';

interface BaseProps {
  children: React.ReactElement;
  colorMap: MapOfIdsToColors;
  open: boolean;
  initialWidth: number;
  onWidthChange: (width: number) => void;
  fixedColumnsCount: number;
  projectId: number;
}

type Props = XOR<{ selectedExperiments: ExperimentWithTrial[] }, { selectedRuns: FlatRun[] }> &
  BaseProps;

const ComparisonView: React.FC<Props> = ({
  children,
  colorMap,
  open,
  initialWidth,
  onWidthChange,
  fixedColumnsCount,
  projectId,
  selectedExperiments,
  selectedRuns,
}) => {
  const scrollbarWidth = useScrollbarWidth();
  const hasPinnedColumns = fixedColumnsCount > 1;
  const isMobile = useMobile();

  const minWidths: [number, number] = useMemo(() => {
    return [fixedColumnsCount * MIN_COLUMN_WIDTH + scrollbarWidth, 100];
  }, [fixedColumnsCount, scrollbarWidth]);

  const trials = useMemo(() => {
    return selectedExperiments?.flatMap((exp) => (exp.bestTrial ? [exp.bestTrial] : [])) ?? [];
  }, [selectedExperiments]);

  const experiments = useMemo(
    () => selectedExperiments?.map((exp) => exp.experiment) ?? [],
    [selectedExperiments],
  );

  const metricData = useMetrics(selectedRuns ?? trials ?? []);

  const tabs: PivotProps['items'] = useMemo(() => {
    return [
      {
        children: selectedRuns ? (
          <CompareMetrics metricData={metricData} selectedRuns={selectedRuns} />
        ) : (
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
        children: selectedRuns ? (
          <CompareHyperparameters
            colorMap={colorMap}
            metricData={metricData}
            projectId={projectId}
            selectedRuns={selectedRuns}
          />
        ) : (
          <CompareHyperparameters
            colorMap={colorMap}
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
        children: selectedRuns ? (
          <TrialsComparisonTable runs={selectedRuns} />
        ) : (
          <TrialsComparisonTable experiment={experiments} trials={trials} />
        ),
        key: 'details',
        label: 'Details',
      },
    ];
  }, [selectedRuns, metricData, selectedExperiments, trials, colorMap, projectId, experiments]);

  const leftPane =
    open && !hasPinnedColumns ? (
      <Message icon="info" title='Pin columns to see them in "Compare View"' />
    ) : (
      children
    );

  const rightPane =
    selectedExperiments?.length === 0 || selectedRuns?.length === 0 ? (
      <Alert description="Select records you would like to compare." message={EMPTY_MESSAGE} />
    ) : (
      <Pivot items={tabs} />
    );

  return (
    <SplitPane
      hidePane={!open ? Pane.Right : isMobile ? Pane.Left : undefined}
      initialWidth={initialWidth}
      leftPane={leftPane}
      minimumWidths={{ left: minWidths[0], right: minWidths[1] }}
      rightPane={rightPane}
      onChange={onWidthChange}
    />
  );
};

export default ComparisonView;
