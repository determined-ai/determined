import { Alert } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { LineChart, Serie } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import LearningCurveChart from 'components/LearningCurveChart';
import Section from 'components/Section';
import TableBatch from 'components/Table/TableBatch';
import { UPlotPoint } from 'components/UPlot/types';
import { terminalRunStates } from 'constants/states';
import useFeature from 'hooks/useFeature';
import { paths } from 'routes/utils';
import { openOrCreateTensorBoard } from 'services/api';
import { V1TrialsSampleResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner/Spinner';
import useUI from 'shared/contexts/stores/UI';
import { glasbeyColor } from 'shared/utils/color';
import { flattenObject, isEqual, isPrimitive } from 'shared/utils/data';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { isNewTabClickEvent, openBlank, routeToReactUrl } from 'shared/utils/routes';
import {
  ExperimentAction as Action,
  CommandResponse,
  ExperimentBase,
  ExperimentSearcherName,
  Hyperparameter,
  HyperparameterType,
  Metric,
  metricTypeParamMap,
  RunState,
  Scale,
} from 'types';
import handleError from 'utils/error';
import { openCommandResponse } from 'utils/wait';

import TrialsComparisonModal from '../TrialsComparisonModal';

import HpTrialTable, { TrialHParams } from './HpTrialTable';
import css from './LearningCurve.module.scss';

interface Props {
  experiment: ExperimentBase;
  filters?: React.ReactNode;
  fullHParams: string[];
  selectedMaxTrial: number;
  selectedMetric: Metric;
  selectedScale: Scale;
}

const MAX_DATAPOINTS = 5000;

export const getCustomSearchVaryingHPs = (
  trialHps: TrialHParams[],
): Record<string, Hyperparameter> => {
  /**
   * For Custom Searchers, add a hyperparameter's column for params that
   * 1) Have more than one unique value (it isn't the same in all trials)
   * 2) Isn't a dictionary of other metrics
   * This is to bypass the need to rely the on the experiment config's
   * definition of hyperparameters and determine what should be shown more dynamically.
   *
   * Note: If we support the other tabs in the future for Custom Searchers
   * such as HpParallelCoordinates, HpScatterPlots, and HpHeatMaps, we will need to
   * generalize this logic a bit.
   */
  const uniq = new Set<string>();
  const check_dict = {} as Record<string, unknown>;
  trialHps.forEach((d) => {
    Object.keys(d.hparams).forEach((key: string) => {
      const value = d.hparams[key];
      if (!(isPrimitive(value) || Array.isArray(value))) {
        /**
         * We have both the flattened and unflattened values in this TrialHParams
         * From `const flatHParams = { ...trial.hparams, ...flattenObject(trial.hparams || {}) };`
         * below in the file. Skip the non flattened dictionaries.
         * Example: {
         *  "dict": { # This is skipped
         *    "key": "value"
         *  },
         *  "dict.key": "value", # This is allowed
         * }
         */
        return;
      }
      if (!(key in check_dict)) {
        check_dict[key] = value;
      } else if (!isEqual(check_dict[key], value)) {
        uniq.add(key);
      }
    });
  });

  // If there's only one result, don't filter by unique results
  const all_keys = trialHps.length === 1 ? Object.keys(check_dict) : Array.from(uniq);
  return all_keys.reduce((acc, key) => {
    acc[key] = {
      type: HyperparameterType.Constant,
    };
    return acc;
  }, {} as Record<string, Hyperparameter>);
};

const LearningCurve: React.FC<Props> = ({
  experiment,
  filters,
  fullHParams,
  selectedMaxTrial,
  selectedMetric,
  selectedScale,
}: Props) => {
  const { ui } = useUI();
  const [trialIds, setTrialIds] = useState<number[]>([]);
  const [batches, setBatches] = useState<number[]>([]);
  const [chartData, setChartData] = useState<(number | null)[][]>([]);
  const [v2ChartData, setV2ChartData] = useState<Serie[]>([]);
  const [trialHps, setTrialHps] = useState<TrialHParams[]>([]);
  const [highlightedTrialId, setHighlightedTrialId] = useState<number>();
  const [hasLoaded, setHasLoaded] = useState(false);
  const [pageError, setPageError] = useState<Error>();
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [showCompareTrials, setShowCompareTrials] = useState(false);
  const chartComponent = useFeature().isOn('chart');

  const hasTrials = trialHps.length !== 0;
  const isExperimentTerminal = terminalRunStates.has(experiment.state as RunState);

  const hyperparameters = useMemo(() => {
    if (experiment.config.searcher.name === ExperimentSearcherName.Custom && trialHps.length > 0) {
      return getCustomSearchVaryingHPs(trialHps);
    } else {
      return fullHParams.reduce((acc, key) => {
        acc[key] = experiment.hyperparameters[key];
        return acc;
      }, {} as Record<string, Hyperparameter>);
    }
  }, [experiment.hyperparameters, fullHParams, trialHps, experiment.config]);

  const handleTrialClick = useCallback(
    (event: MouseEvent, trialId: number) => {
      const href = paths.trialDetails(trialId, experiment.id);
      if (isNewTabClickEvent(event)) openBlank(href);
      else routeToReactUrl(href);
    },
    [experiment.id],
  );

  const handleTrialFocus = useCallback((trialId: number | null) => {
    setHighlightedTrialId(trialId != null ? trialId : undefined);
  }, []);

  const handlePointClick = useCallback(
    (e: MouseEvent, point: UPlotPoint) => {
      const trialId = point ? trialIds[point.seriesIdx - 1] : undefined;
      if (trialId) handleTrialClick(e, trialId);
    },
    [handleTrialClick, trialIds],
  );

  const handlePointFocus = useCallback(
    (point?: UPlotPoint) => {
      const trialId = point ? trialIds[point.seriesIdx - 1] : undefined;
      if (trialId) handleTrialFocus(trialId);
    },
    [handleTrialFocus, trialIds],
  );

  const handleTableMouseEnter = useCallback((event: React.MouseEvent, record: TrialHParams) => {
    if (record.id) setHighlightedTrialId(record.id);
  }, []);

  const handleTableMouseLeave = useCallback(() => {
    setHighlightedTrialId(undefined);
  }, []);

  const clearSelected = useCallback(() => {
    setSelectedRowKeys([]);
  }, []);

  useEffect(() => {
    if (ui.isPageHidden) return;

    const canceler = new AbortController();
    const trialIdsMap: Record<number, number> = {};
    const trialDataMap: Record<number, number[]> = {};
    const trialHpMap: Record<number, TrialHParams> = {};
    const batchesMap: Record<number, number> = {};
    const metricsMap: Record<number, Record<number, number>> = {};
    const v2MetricsMap: Record<number, [number, number][]> = {};

    setHasLoaded(false);

    readStream<V1TrialsSampleResponse>(
      detApi.StreamingInternal.trialsSample(
        experiment.id,
        selectedMetric.name,
        metricTypeParamMap[selectedMetric.type],
        selectedMaxTrial,
        MAX_DATAPOINTS,
        undefined,
        undefined,
        undefined,
        { signal: canceler.signal },
      ),
      (event) => {
        if (!event?.trials || !Array.isArray(event.trials)) return;

        /*
         * Cache trial ids, hparams, and metric values into easily searchable
         * dictionaries, then construct the necessary data structures to render the
         * chart and the table.
         */

        (event.promotedTrials || []).forEach((trialId) => (trialIdsMap[trialId] = trialId));
        (event.demotedTrials || []).forEach((trialId) => delete trialIdsMap[trialId]);
        const newTrialIds = Object.values(trialIdsMap);
        setTrialIds(newTrialIds);

        (event.trials || []).forEach((trial) => {
          const id = trial.trialId;

          // This allows for both typical nested hyperparameters and nested categorgical
          // hyperparameter values to be shown, with HpTrialTable deciding which are displayed.
          const flatHParams = { ...trial.hparams, ...flattenObject(trial.hparams || {}) };

          const hasHParams = Object.keys(flatHParams).length !== 0;

          if (hasHParams && !trialHpMap[id]) {
            trialHpMap[id] = { hparams: flatHParams, id, metric: null };
          }

          trialDataMap[id] = trialDataMap[id] || [];
          metricsMap[id] = metricsMap[id] || {};
          v2MetricsMap[id] = [];

          trial.data.forEach((datapoint) => {
            batchesMap[datapoint.batches] = datapoint.batches;
            metricsMap[id][datapoint.batches] = datapoint.value;
            v2MetricsMap[id].push([datapoint.batches, datapoint.value]);
            trialHpMap[id].metric = datapoint.value;
          });
        });

        const newTrialHps = newTrialIds.map((id) => trialHpMap[id]);
        setTrialHps(newTrialHps);

        const newBatches = Object.values(batchesMap);
        setBatches(newBatches);

        const newChartData = newTrialIds.map((trialId) =>
          newBatches.map((batch) => {
            /**
             * TODO: filtering NaN, +/- Infinity for now, but handle it later with
             * dynamic min/max ranges via uPlot.Scales.
             */
            const value = metricsMap[trialId][batch];
            return Number.isFinite(value) ? value : null;
          }),
        );
        setChartData(newChartData);

        const v2NewChartData = newTrialIds
          .filter((trialId) => !selectedRowKeys.length || selectedRowKeys.includes(trialId))
          .map((trialId) => ({
            color: glasbeyColor(trialId),
            data: { [XAxisDomain.Batches]: v2MetricsMap[trialId] },
            key: trialId,
            name: `trial ${trialId}`,
          }));
        setV2ChartData(v2NewChartData);

        // One successful event as come through.
        setHasLoaded(true);
      },
      (e) => {
        setPageError(e);
        setHasLoaded(true);
      },
    );

    return () => canceler.abort();
  }, [experiment.id, selectedMaxTrial, selectedMetric, selectedRowKeys, ui.isPageHidden]);

  const sendBatchActions = useCallback(
    async (action: Action) => {
      if (action === Action.OpenTensorBoard) {
        return await openOrCreateTensorBoard({
          trialIds: selectedRowKeys,
          workspaceId: experiment.workspaceId,
        });
      } else if (action === Action.CompareTrials) {
        return setShowCompareTrials(true);
      }
    },
    [selectedRowKeys, experiment],
  );

  const submitBatchAction = useCallback(
    async (action: Action) => {
      try {
        const result = await sendBatchActions(action);
        if (action === Action.OpenTensorBoard && result) {
          openCommandResponse(result as CommandResponse);
        }
      } catch (e) {
        const publicSubject =
          action === Action.OpenTensorBoard
            ? 'Unable to View TensorBoard for Selected Trials'
            : `Unable to ${action} Selected Trials`;
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject,
          silent: false,
          type: ErrorType.Server,
        });
      }
    },
    [sendBatchActions],
  );

  const handleTableRowSelect = useCallback(
    (rowKeys: unknown) => setSelectedRowKeys(rowKeys as number[]),
    [],
  );

  const handleTrialUnselect = useCallback(
    (trialId: number) => setSelectedRowKeys((rowKeys) => rowKeys.filter((id) => id !== trialId)),
    [],
  );

  if (pageError) {
    return <Message title={pageError.message} />;
  } else if (hasLoaded && !hasTrials) {
    return isExperimentTerminal ? (
      <Message title="No learning curve data to show." type={MessageType.Empty} />
    ) : (
      <div className={css.waiting}>
        <Alert
          description="Please wait until the experiment is further along."
          message="Not enough data points to plot."
        />
        <Spinner />
      </div>
    );
  }

  return (
    <div className={css.base}>
      <Section bodyBorder bodyScroll filters={filters} loading={!hasLoaded}>
        <div className={css.container}>
          <div className={css.chart}>
            {chartComponent ? (
              <LineChart
                focusedSeries={highlightedTrialId && trialIds.indexOf(highlightedTrialId)}
                scale={selectedScale}
                series={v2ChartData}
                xLabel="Batches Processed"
                yLabel={`[${selectedMetric.type[0].toUpperCase()}] ${selectedMetric.name}`}
                onPointClick={handlePointClick}
                onPointFocus={handlePointFocus}
              />
            ) : (
              <LearningCurveChart
                data={chartData}
                focusedTrialId={highlightedTrialId}
                selectedMetric={selectedMetric}
                selectedScale={selectedScale}
                selectedTrialIds={selectedRowKeys}
                trialIds={trialIds}
                xValues={batches}
                onTrialClick={handleTrialClick}
                onTrialFocus={handleTrialFocus}
              />
            )}
          </div>
          <TableBatch
            actions={[
              { label: Action.OpenTensorBoard, value: Action.OpenTensorBoard },
              { label: Action.CompareTrials, value: Action.CompareTrials },
            ]}
            selectedRowCount={selectedRowKeys.length}
            onAction={(action) => submitBatchAction(action as Action)}
            onClear={clearSelected}
          />
          <HpTrialTable
            experimentId={experiment.id}
            handleTableRowSelect={handleTableRowSelect}
            highlightedTrialId={highlightedTrialId}
            hyperparameters={hyperparameters}
            metric={selectedMetric}
            selectedRowKeys={selectedRowKeys}
            selection={true}
            trialHps={trialHps}
            onMouseEnter={handleTableMouseEnter}
            onMouseLeave={handleTableMouseLeave}
          />
        </div>
      </Section>
      {showCompareTrials && (
        <TrialsComparisonModal
          experiment={experiment}
          trials={selectedRowKeys}
          visible={showCompareTrials}
          onCancel={() => setShowCompareTrials(false)}
          onUnselect={handleTrialUnselect}
        />
      )}
    </div>
  );
};

export default LearningCurve;
