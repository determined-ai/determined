import React, { useEffect, useRef } from 'react';

import { useSetDynamicTabBar } from 'components/DynamicTabs';
import Grid, { GridMode } from 'components/Grid';
import LearningCurveChart from 'components/LearningCurveChart';
import Page from 'components/Page';
import Section from 'components/Section';
import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { SyncProvider } from 'components/UPlot/SyncableBounds';
import useSettings from 'hooks/useSettings';
import TrialTable from 'pages/TrialsComparison/Table/TrialTable';
import { V1AugmentedTrial, V1OrderBy } from 'services/api-ts-sdk';
import { Scale } from 'types';
import { metricToKey } from 'utils/metric';

import useHighlight from '../../hooks/useHighlight';

import useTrialActions from './Actions/useTrialActions';
import {
  useTrialCollections,
} from './Collections/useTrialCollections';
import useLearningCurveData from './Metrics/useLearningCurveData';
import { trialsTableSettingsConfig } from './Table/settings';
import { useFetchTrials } from './Trials/useFetchTrials';
import css from './TrialsComparison.module.scss';

const initData = [ [] ];
interface Props {
  projectId: string;
}

const TrialsComparison: React.FC<Props> = ({ projectId }) => {

  const tableSettingsHook = useSettings<InteractiveTableSettings>(trialsTableSettingsConfig);
  const { settings: tableSettings } = tableSettingsHook;

  const C = useTrialCollections(projectId, tableSettings);

  const trials = useFetchTrials({
    filters: C.filters,
    limit: tableSettings.tableLimit,
    offset: tableSettings.tableOffset,
    sorter: C.sorter,
  });

  const A = useTrialActions({
    filters: C.filters,
    openCreateModal: C.openCreateModal,
    sorter: C.sorter,
  });

  const highlights = useHighlight((trial: V1AugmentedTrial): number => trial.trialId);

  const containerRef = useRef<HTMLElement>(null);

  const chartSeries = useLearningCurveData(trials.ids, trials.metrics, trials.maxBatch);

  useSetDynamicTabBar(C.controls);

  return (
    <Page className={css.base} containerRef={containerRef}>
      <Section
        bodyBorder
        bodyScroll>
        <div className={css.container}>
          <div className={css.chart}>
            {/* <Grid
              border={true}
              // need to use screen size
              minItemWidth={600}
              mode={GridMode.AutoFill}>
              <SyncProvider>
                {chartSeries && trials.metrics.map((metric) => (
                  <LearningCurveChart
                    data={chartSeries.metrics?.[metricToKey(metric)] ?? initData}
                    focusedTrialId={highlights.id}
                    key={metricToKey(metric)}
                    selectedMetric={metric}
                    selectedScale={Scale.Linear}
                    selectedTrialIds={A.selectedTrials}
                    trialIds={trials.ids}
                    xValues={chartSeries.batches}
                    onTrialFocus={highlights.focus}
                  />

                )) }
              </SyncProvider>
            </Grid> */}
          </div>
          {A.dispatcher}
          <TrialTable
            actionsInterface={A}
            collectionsInterface={C}
            containerRef={containerRef}
            highlights={highlights}
            tableSettingsHook={tableSettingsHook}
            trialsWithMetadata={trials}
          />
        </div>
      </Section>
      {A.modalContextHolder}
      {C.modalContextHolder}
    </Page>

  );
};

export default TrialsComparison;
