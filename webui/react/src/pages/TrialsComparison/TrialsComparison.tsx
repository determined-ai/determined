import React, { useCallback, useMemo, useRef } from 'react';
import { debounce } from 'throttle-debounce';

import { useSetDynamicTabBar } from 'components/DynamicTabs';
import Grid, { GridMode } from 'components/Grid';
import Empty from 'components/kit/Empty';
import LearningCurveChart from 'components/LearningCurveChart';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import Section from 'components/Section';
import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { SyncProvider } from 'components/UPlot/SyncProvider';
import { useSettings } from 'hooks/useSettings';
import TrialTable from 'pages/TrialsComparison/Table/TrialTable';
import { V1AugmentedTrial } from 'services/api-ts-sdk';
import { Scale } from 'types';
import { metricToKey } from 'utils/metric';
import { intersection } from 'utils/set';

import useHighlight from '../../hooks/useHighlight';

import useTrialActions from './Actions/useTrialActions';
import { useTrialCollections } from './Collections/useTrialCollections';
import useLearningCurveData from './Metrics/useLearningCurveData';
import { trialsTableSettingsConfig } from './Table/settings';
import { useFetchTrials } from './Trials/useFetchTrials';
import css from './TrialsComparison.module.scss';

interface Props {
  projectId: string;
  workspaceId: number;
}

const TrialsComparison: React.FC<Props> = ({ projectId, workspaceId }) => {
  const config = useMemo(() => trialsTableSettingsConfig(projectId), [projectId]);
  const tableSettingsHook = useSettings<InteractiveTableSettings>(config);

  const collections = useTrialCollections(projectId, tableSettingsHook);

  const { settings: tableSettings } = tableSettingsHook;

  const { trials, refetch, loading } = useFetchTrials({
    filters: collections.filters,
    limit: tableSettings.tableLimit || 0,
    offset: tableSettings.tableOffset || 0,
    sorter: collections.sorter,
  });

  const actions = useTrialActions({
    availableIds: trials.ids,
    filters: collections.filters,
    openCreateModal: collections.openCreateModal,
    refetch,
    sorter: collections.sorter,
    workspaceId,
  });

  const highlights = useHighlight((trial: V1AugmentedTrial): number => trial.trialId);

  const containerRef = useRef<HTMLElement>(null);

  const learningCurveData = useLearningCurveData(trials.ids, trials.metrics, trials.maxBatch);

  useSetDynamicTabBar(location.pathname.includes('trials') ? collections.controls : undefined);

  const handleClickFirstFive = useCallback(() => {
    actions.selectTrials(trials.ids.slice(0, 5));
  }, [actions, trials.ids]);

  const { setSelectedTrials } = actions;
  const handleTrialClick = useCallback(
    (_: MouseEvent, trialId: number) => {
      setSelectedTrials((ids) => {
        const otherIds = ids.filter((id) => id !== trialId);
        const trialIncluded = ids.includes(trialId);
        return trialIncluded ? otherIds : [...otherIds, trialId];
      });
    },
    [setSelectedTrials],
  );

  const handleTrialFocus = useMemo(() => debounce(1000, highlights.focus), [highlights.focus]);

  return (
    <Page breadcrumb={[]} containerRef={containerRef}>
      <div className={css.base}>
        <Section bodyBorder bodyScroll>
          <div className={css.container}>
            <div className={css.chart}>
              {actions.selectedTrials.length === 0 ? (
                <Empty
                  description={
                    <>
                      Choose trials to plot or{' '}
                      <a onClick={handleClickFirstFive}>select first five</a>
                    </>
                  }
                  icon="experiment"
                  title="No Trials Selected"
                />
              ) : trials.metrics.length === 0 ? (
                <Message title="No Metrics for Selected Trials" type={MessageType.Empty} />
              ) : (
                <Grid
                  border={true}
                  //  TODO: use screen size
                  minItemWidth={600}
                  mode={GridMode.AutoFill}>
                  <SyncProvider>
                    {trials.metrics.map((metric) => {
                      const metricKey = metricToKey(metric);
                      const metricInfo = learningCurveData?.infoForMetrics?.[metricKey];
                      const nonEmptyTrials = metricInfo?.nonEmptyTrials;
                      const selectedTrials = new Set(actions.selectedTrials);
                      const hasData =
                        nonEmptyTrials && intersection(selectedTrials, nonEmptyTrials).size > 0;
                      if (!hasData)
                        return (
                          <Message
                            key={metricKey}
                            title={`No ${metric.type} ${metric.name} data for selected trials`}
                            type={MessageType.Empty}
                          />
                        );
                      return (
                        <LearningCurveChart
                          data={metricInfo.chartData}
                          focusedTrialId={highlights.id}
                          key={metricKey}
                          selectedMetric={metric}
                          selectedScale={Scale.Linear}
                          selectedTrialIds={actions.selectedTrials}
                          trialIds={trials.ids}
                          xValues={learningCurveData?.batches ?? []}
                          onTrialClick={handleTrialClick}
                          onTrialFocus={handleTrialFocus}
                        />
                      );
                    })}
                  </SyncProvider>
                </Grid>
              )}
            </div>
            <div className={css.table}>
              {actions.dispatcher}
              <TrialTable
                actionsInterface={actions}
                collectionsInterface={collections}
                containerRef={containerRef}
                highlights={highlights}
                loading={loading}
                tableSettingsHook={tableSettingsHook}
                trialsWithMetadata={trials}
              />
            </div>
          </div>
        </Section>
        {actions.modalContextHolder}
      </div>
    </Page>
  );
};

export default TrialsComparison;
