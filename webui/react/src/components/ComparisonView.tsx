import Alert from 'hew/Alert';
import { MIN_COLUMN_WIDTH } from 'hew/DataGrid/columns';
import Message from 'hew/Message';
import Pivot, { PivotProps } from 'hew/Pivot';
import Spinner from 'hew/Spinner';
import SplitPane, { Pane } from 'hew/SplitPane';
import { Loadable, NotLoaded } from 'hew/utils/loadable';
import React, { useMemo } from 'react';

import CompareHyperparameters from 'components/CompareHyperparameters';
import { useAsync } from 'hooks/useAsync';
import { MapOfIdsToColors } from 'hooks/useGlasbey';
import { useMetrics } from 'hooks/useMetrics';
import useMobile from 'hooks/useMobile';
import useScrollbarWidth from 'hooks/useScrollbarWidth';
import { TrialsComparisonTable } from 'pages/ExperimentDetails/TrialsComparisonModal';
import { searchExperiments, searchRuns } from 'services/api';
import { ExperimentWithTrial, FlatRun, XOR } from 'types';
import handleError from 'utils/error';
import { getIdsFilter as getExperimentIdsFilter } from 'utils/experiment';
import { getIdsFilter as getRunIdsFilter } from 'utils/flatRun';

import CompareMetrics from './CompareMetrics';
import { INIT_FORMSET } from './FilterForm/components/FilterFormStore';

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

type Props = XOR<{ selectedExperimentIds: number[] }, { selectedRunIds: number[] }> & BaseProps;

const ComparisonView: React.FC<Props> = ({
  children,
  colorMap,
  open,
  initialWidth,
  onWidthChange,
  fixedColumnsCount,
  projectId,
  selectedExperimentIds,
  selectedRunIds,
}) => {
  const scrollbarWidth = useScrollbarWidth();
  const hasPinnedColumns = fixedColumnsCount > 1;
  const isMobile = useMobile();

  const loadableSelectedExperiments = useAsync(async () => {
    if (selectedExperimentIds?.length) {
      const filterFormSet = INIT_FORMSET;
      try {
        const filter = getExperimentIdsFilter(filterFormSet, {
          selections: selectedExperimentIds,
          type: 'ONLY_IN',
        });
        const response = await searchExperiments({
          filter: JSON.stringify(filter),
          limit: 50,
        });
        return response.experiments;
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to fetch experiments for comparison' });
        return NotLoaded;
      }
    }
    return NotLoaded;
  }, [selectedExperimentIds]);

  const selectedExperiments: ExperimentWithTrial[] | undefined = Loadable.getOrElse(
    undefined,
    loadableSelectedExperiments,
  );

  const loadableSelectedRuns = useAsync(async () => {
    if (selectedRunIds?.length) {
      const filterFormSet = INIT_FORMSET;
      try {
        const filter = getRunIdsFilter(filterFormSet, {
          selections: selectedRunIds,
          type: 'ONLY_IN',
        });
        const response = await searchRuns({
          filter: JSON.stringify(filter),
          limit: 50,
        });
        return response.runs;
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to fetch runs for comparison' });
        return NotLoaded;
      }
    }
    return NotLoaded;
  }, [selectedRunIds]);

  const selectedRuns: FlatRun[] | undefined = Loadable.getOrElse(undefined, loadableSelectedRuns);

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
            selectedExperiments={selectedExperiments ?? []}
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
            selectedExperiments={selectedExperiments ?? []}
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
    (selectedExperimentIds?.length && selectedExperiments === undefined) ||
    (selectedRunIds?.length && selectedRuns === undefined) ? (
      <Spinner />
    ) : selectedExperimentIds?.length === 0 || selectedRunIds?.length === 0 ? (
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
