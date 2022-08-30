import yaml from 'js-yaml';
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
import { V1AugmentedTrial } from 'services/api-ts-sdk';
import Message, { MessageType } from 'shared/components/Message';
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

interface Props {
  projectId: string;
}

const TrialsComparison: React.FC<Props> = ({ projectId }) => {
  const tableSettingsHook = useSettings<InteractiveTableSettings>(trialsTableSettingsConfig);

  const refetcher = useRef<() => void>();

  const collections = useTrialCollections(projectId, tableSettingsHook, refetcher);

  const { settings: tableSettings } = tableSettingsHook;

  const { trials, refetch } = useFetchTrials({
    filters: collections.filters,
    limit: tableSettings.tableLimit,
    offset: tableSettings.tableOffset,
    sorter: collections.sorter,
  });

  useEffect(() => refetcher.current = refetch, [ refetch ]);

  const actions = useTrialActions({
    filters: collections.filters,
    openCreateModal: collections.openCreateModal,
    refetch,
    sorter: collections.sorter,
  });

  const highlights = useHighlight((trial: V1AugmentedTrial): number => trial.trialId);

  const containerRef = useRef<HTMLElement>(null);

  const chartSeries = useLearningCurveData(trials.ids, trials.metrics, trials.maxBatch);

  useSetDynamicTabBar(collections.controls);

  return (
    <Page className={css.base} containerRef={containerRef}>
      <Section bodyBorder bodyScroll>
        <div className={css.container}>
          <div className={css.chart}>
            {trials.metrics.length === 0 ? (
              <Message title="No Metrics for Selected Trials" type={MessageType.Empty} />
            ) : (
              <Grid
                border={true}
                //  TODO: use screen size
                minItemWidth={600}
                mode={GridMode.AutoFill}>
                <SyncProvider>
                  {trials.metrics.map((metric) => (
                    <LearningCurveChart
                      data={chartSeries?.metrics?.[metricToKey(metric)] ?? [ [] ]}
                      focusedTrialId={highlights.id}
                      key={metricToKey(metric)}
                      selectedMetric={metric}
                      selectedScale={Scale.Linear}
                      selectedTrialIds={actions.selectedTrials}
                      trialIds={trials.ids}
                      xValues={chartSeries?.batches ?? []}
                      onTrialFocus={highlights.focus}
                    />
                  ))}
                </SyncProvider>
              </Grid>
            )}
          </div>
          {actions.dispatcher}
          <TrialTable
            actionsInterface={actions}
            collectionsInterface={collections}
            containerRef={containerRef}
            highlights={highlights}
            tableSettingsHook={tableSettingsHook}
            trialsWithMetadata={trials}
          />
        </div>
      </Section>
      {actions.modalContextHolder}
      {collections.modalContextHolder}
    </Page>
  );
};

export default TrialsComparison;
