import Alert from 'hew/Alert';
import { MIN_COLUMN_WIDTH } from 'hew/DataGrid/columns';
import Message from 'hew/Message';
import Pivot, { PivotProps } from 'hew/Pivot';
import Spinner from 'hew/Spinner';
import SplitPane, { Pane } from 'hew/SplitPane';
import { Loadable, NotLoaded } from 'hew/utils/loadable';
import React, { useMemo, useState } from 'react';

import CompareHyperparameters from 'components/CompareHyperparameters';
import { useAsync } from 'hooks/useAsync';
import { MapOfIdsToColors } from 'hooks/useGlasbey';
import { useMetrics } from 'hooks/useMetrics';
import useMobile from 'hooks/useMobile';
import useScrollbarWidth from 'hooks/useScrollbarWidth';
import { TrialsComparisonTable } from 'pages/ExperimentDetails/TrialsComparisonModal';
import { searchExperiments, searchRuns } from 'services/api';
import { V1ColumnType, V1LocationType } from 'services/api-ts-sdk';
import { ExperimentWithTrial, FlatRun, SelectionType, XOR } from 'types';
import handleError from 'utils/error';
import { getIdsFilter as getExperimentIdsFilter } from 'utils/experiment';
import { combine } from 'utils/filterFormSet';
import { getIdsFilter as getRunIdsFilter } from 'utils/flatRun';

import CompareMetrics from './CompareMetrics';
import { INIT_FORMSET } from './FilterForm/components/FilterFormStore';
import { FilterFormSet, Operator } from './FilterForm/components/type';

export const EMPTY_MESSAGE = 'No items selected.';

interface BaseProps {
  children: React.ReactElement;
  colorMap: MapOfIdsToColors;
  open: boolean;
  initialWidth: number;
  onWidthChange: (width: number) => void;
  fixedColumnsCount: number;
  projectId: number;
  searchId?: number;
  tableFilters: string;
}

type Props = XOR<{ experimentSelection: SelectionType }, { runSelection: SelectionType }> &
  BaseProps;

const SELECTION_LIMIT = 50;

interface TabsProps {
  colorMap: MapOfIdsToColors;
  loadableSelectedExperiments: Loadable<ExperimentWithTrial[]>;
  loadableSelectedRuns: Loadable<FlatRun[]>;
  projectId: number;
}

const Tabs = ({
  colorMap,
  loadableSelectedExperiments,
  loadableSelectedRuns,
  projectId,
}: TabsProps) => {
  const selectedExperiments: ExperimentWithTrial[] | undefined = Loadable.getOrElse(
    undefined,
    loadableSelectedExperiments,
  );

  const selectedRuns: FlatRun[] | undefined = Loadable.getOrElse(undefined, loadableSelectedRuns);

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
          <CompareMetrics colorMap={colorMap} metricData={metricData} selectedRuns={selectedRuns} />
        ) : (
          <CompareMetrics
            colorMap={colorMap}
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

  return <Pivot items={tabs} />;
};

const ComparisonView: React.FC<Props> = ({
  children,
  colorMap,
  open,
  initialWidth,
  onWidthChange,
  fixedColumnsCount,
  projectId,
  experimentSelection,
  runSelection,
  searchId,
  tableFilters,
}) => {
  const scrollbarWidth = useScrollbarWidth();
  const hasPinnedColumns = fixedColumnsCount > 1;
  const isMobile = useMobile();

  const [isSelectionLimitReached, setIsSelectionLimitReached] = useState(false);

  const loadableSelectedExperiments = useAsync(async () => {
    if (
      !open ||
      !experimentSelection ||
      (experimentSelection.type === 'ONLY_IN' && experimentSelection.selections.length === 0)
    ) {
      return NotLoaded;
    }
    try {
      const filters = JSON.parse(tableFilters) as FilterFormSet;
      const filterFormSet = INIT_FORMSET;
      const filter = getExperimentIdsFilter(filterFormSet, experimentSelection);
      if (experimentSelection.type === 'ALL_EXCEPT') {
        filter.filterGroup = combine(filter.filterGroup, 'and', filters.filterGroup);
      }
      const response = await searchExperiments({
        filter: JSON.stringify(filter),
        limit: SELECTION_LIMIT,
      });
      setIsSelectionLimitReached(
        !!response?.pagination?.total && response?.pagination?.total > SELECTION_LIMIT,
      );
      return response.experiments;
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch experiments for comparison' });
      return NotLoaded;
    }
  }, [experimentSelection, open, tableFilters]);

  const loadableSelectedRuns = useAsync(async () => {
    if (
      !open ||
      !runSelection ||
      (runSelection.type === 'ONLY_IN' && runSelection.selections.length === 0)
    ) {
      return NotLoaded;
    }
    const filterFormSet = INIT_FORMSET;
    try {
      const filters = JSON.parse(tableFilters) as FilterFormSet;
      const filter = getRunIdsFilter(filterFormSet, runSelection);
      if (searchId) {
        // only display trials for search
        const searchFilter = {
          columnName: 'experimentId',
          kind: 'field' as const,
          location: V1LocationType.RUN,
          operator: Operator.Eq,
          type: V1ColumnType.NUMBER,
          value: searchId,
        };
        filter.filterGroup = combine(filter.filterGroup, 'and', searchFilter);
      }
      if (runSelection.type === 'ALL_EXCEPT') {
        filter.filterGroup = combine(filter.filterGroup, 'and', filters.filterGroup);
      }
      const response = await searchRuns({
        filter: JSON.stringify(filter),
        limit: SELECTION_LIMIT,
        projectId,
      });
      setIsSelectionLimitReached(
        !!response?.pagination?.total && response?.pagination?.total > SELECTION_LIMIT,
      );
      return response.runs;
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch runs for comparison' });
      return NotLoaded;
    }
  }, [open, projectId, runSelection, searchId, tableFilters]);

  const minWidths: [number, number] = useMemo(() => {
    return [fixedColumnsCount * MIN_COLUMN_WIDTH + scrollbarWidth, 100];
  }, [fixedColumnsCount, scrollbarWidth]);

  const leftPane =
    open && !hasPinnedColumns ? (
      <Message icon="info" title='Pin columns to see them in "Compare View"' />
    ) : (
      children
    );

  const getRightPaneContent = () => {
    if (experimentSelection) {
      if (experimentSelection.type === 'ONLY_IN' && experimentSelection.selections.length === 0) {
        return (
          <Alert description="Select records you would like to compare." message={EMPTY_MESSAGE} />
        );
      }
      if (loadableSelectedExperiments.isNotLoaded) {
        return <Spinner spinning />;
      }
    }
    if (runSelection) {
      if (runSelection.type === 'ONLY_IN' && runSelection.selections.length === 0) {
        return (
          <Alert description="Select records you would like to compare." message={EMPTY_MESSAGE} />
        );
      }
      if (loadableSelectedRuns.isNotLoaded) {
        return <Spinner spinning />;
      }
    }
    return (
      <>
        {isSelectionLimitReached && (
          <Alert message={`Only up to ${SELECTION_LIMIT} records can be compared`} />
        )}
        <Tabs
          colorMap={colorMap}
          loadableSelectedExperiments={loadableSelectedExperiments}
          loadableSelectedRuns={loadableSelectedRuns}
          projectId={projectId}
        />
      </>
    );
  };

  return (
    <SplitPane
      hidePane={!open ? Pane.Right : isMobile ? Pane.Left : undefined}
      initialWidth={initialWidth}
      leftPane={leftPane}
      minimumWidths={{ left: minWidths[0], right: minWidths[1] }}
      rightPane={getRightPaneContent()}
      onChange={onWidthChange}
    />
  );
};

export default ComparisonView;
