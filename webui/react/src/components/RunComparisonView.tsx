import Alert from 'hew/Alert';
import { MIN_COLUMN_WIDTH } from 'hew/DataGrid/columns';
import Message from 'hew/Message';
import Pivot, { PivotProps } from 'hew/Pivot';
import SplitPane, { Pane } from 'hew/SplitPane';
import React, { useMemo } from 'react';

import CompareHyperparameters from 'components/CompareHyperparameters';
import { useMetrics } from 'hooks/useMetrics';
import useMobile from 'hooks/useMobile';
import useScrollbarWidth from 'hooks/useScrollbarWidth';
import { TrialsComparisonTable } from 'pages/ExperimentDetails/TrialsComparisonModal';
import { FlatRun } from 'types';

import CompareMetrics from './CompareMetrics';

interface Props {
  children: React.ReactElement;
  open: boolean;
  initialWidth: number;
  onWidthChange: (width: number) => void;
  fixedColumnsCount: number;
  projectId: number;
  selectedRuns: FlatRun[];
}

const RunComparisonView: React.FC<Props> = ({
  children,
  open,
  initialWidth,
  onWidthChange,
  fixedColumnsCount,
  projectId,
  selectedRuns,
}) => {
  const scrollbarWidth = useScrollbarWidth();
  const hasPinnedColumns = fixedColumnsCount > 1;
  const isMobile = useMobile();

  const minWidths: [number, number] = useMemo(() => {
    return [fixedColumnsCount * MIN_COLUMN_WIDTH + scrollbarWidth, 100];
  }, [fixedColumnsCount, scrollbarWidth]);

  const metricData = useMetrics(selectedRuns);

  const tabs: PivotProps['items'] = useMemo(() => {
    return [
      {
        children: <CompareMetrics metricData={metricData} selectedRuns={selectedRuns} />,
        key: 'metrics',
        label: 'Metrics',
      },
      {
        children: (
          <CompareHyperparameters
            metricData={metricData}
            projectId={projectId}
            selectedRuns={selectedRuns}
          />
        ),
        key: 'hyperparameters',
        label: 'Hyperparameters',
      },
      {
        children: <TrialsComparisonTable runs={selectedRuns} />,
        key: 'details',
        label: 'Details',
      },
    ];
  }, [metricData, projectId, selectedRuns]);

  const leftPane =
    open && !hasPinnedColumns ? (
      <Message icon="info" title='Pin columns to see them in "Compare View"' />
    ) : (
      children
    );

  const rightPane =
    selectedRuns.length === 0 ? (
      <Alert description="Select runs you would like to compare." message="No runs selected." />
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

export default RunComparisonView;
