import { Alert } from 'antd';
import Hermes, { DimensionType } from 'hermes-parallel-coordinates';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Empty from 'components/kit/Empty';
import ParallelCoordinates from 'components/ParallelCoordinates';
import Section from 'components/Section';
import Spinner from 'components/Spinner/Spinner';
import TableBatch from 'components/Table/TableBatch';
import { terminalRunStates } from 'constants/states';
import { openOrCreateTensorBoard } from 'services/api';
import { V1TrialsSnapshotResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import useUI from 'stores/contexts/UI';
import { Primitive, Range } from 'types';
import {
  ExperimentAction as Action,
  CommandResponse,
  ExperimentBase,
  HpTrialData,
  Hyperparameter,
  HyperparameterType,
  Metric,
  MetricType,
  metricTypeParamMap,
  Scale,
  TrialDetails,
} from 'types';
import { defaultNumericRange, getColorScale, getNumericRange, updateRange } from 'utils/chart';
import { rgba2str, str2rgba } from 'utils/color';
import { clone, flattenObject, isPrimitive } from 'utils/data';
import { ErrorLevel, ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { metricToStr } from 'utils/metric';
import { numericSorter } from 'utils/sort';
import { openCommandResponse } from 'utils/wait';

import TrialsComparisonModal from '../TrialsComparisonModal';

import css from './HpParallelCoordinates.module.scss';
import HpTrialTable, { TrialHParams } from './HpTrialTable';

interface Props {
  experiment: ExperimentBase;
  filters?: React.ReactNode;
  fullHParams: string[];
  selectedBatch: number;
  selectedBatchMargin: number;
  selectedHParams: string[];
  selectedMetric?: Metric;
  selectedScale: Scale;
  focusedTrial?: TrialDetails;
}

const HpParallelCoordinates: React.FC<Props> = ({
  experiment,
  filters,
  fullHParams,
  selectedBatch,
  selectedBatchMargin,
  selectedHParams,
  selectedMetric,
  selectedScale,
  focusedTrial,
}: Props) => {
  const { ui } = useUI();
  const tooltipRef = useRef<HTMLDivElement>(null);
  const trialIdRef = useRef<HTMLDivElement>(null);
  const metricValueRef = useRef<HTMLDivElement>(null);
  const [hasLoaded, setHasLoaded] = useState(false);
  const [chartData, setChartData] = useState<HpTrialData>();
  const [trialHps, setTrialHps] = useState<TrialHParams[]>([]);
  const [pageError, setPageError] = useState<Error>();
  const [filteredTrialIdMap, setFilteredTrialIdMap] = useState<Record<number, boolean>>();
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [showCompareTrials, setShowCompareTrials] = useState(false);
  const [hermesCreatedFilters, setHermesCreatedFilters] = useState<Hermes.Filters>({});

  const hyperparameters = useMemo(() => {
    return fullHParams.reduce((acc, key) => {
      acc[key] = experiment.hyperparameters[key];
      return acc;
    }, {} as Record<string, Hyperparameter>);
  }, [experiment.hyperparameters, fullHParams]);

  const isExperimentTerminal = terminalRunStates.has(experiment.state);

  const smallerIsBetter = useMemo(() => {
    if (
      selectedMetric &&
      selectedMetric.type === MetricType.Validation &&
      selectedMetric.name === experiment.config.searcher.metric
    ) {
      return experiment.config.searcher.smallerIsBetter;
    }
    return undefined;
  }, [experiment.config.searcher, selectedMetric]);

  const resetFilteredTrials = useCallback(() => {
    // Skip if there isn't any chart data.
    if (!chartData) return;

    // Initialize a new trial id filter map.
    const newFilteredTrialIdMap = chartData.trialIds.reduce((acc, trialId) => {
      acc[trialId] = true;
      return acc;
    }, {} as Record<number, boolean>);

    // Figure out which trials are filtered out based on user filters.
    Object.entries(hermesCreatedFilters).forEach(([key, list]) => {
      if (!chartData.data[key] || list.length === 0) return;

      chartData.data[key].forEach((value, index) => {
        let isWithinFilter = false;

        list.forEach((filter: Hermes.Filter) => {
          const min = Math.min(Number(filter.value0), Number(filter.value1));
          const max = Math.max(Number(filter.value0), Number(filter.value1));
          if (Number(value) >= min && Number(value) <= max) {
            isWithinFilter = true;
          }
        });

        if (!isWithinFilter) {
          const trialId = chartData.trialIds[index];
          newFilteredTrialIdMap[trialId] = false;
        }
      });
    });

    setFilteredTrialIdMap(newFilteredTrialIdMap);
  }, [chartData, hermesCreatedFilters]);

  useEffect(() => {
    resetFilteredTrials();
  }, [resetFilteredTrials]);

  const colorScale = useMemo(() => {
    return getColorScale(ui.theme, chartData?.metricRange, smallerIsBetter);
  }, [chartData?.metricRange, smallerIsBetter, ui.theme]);

  const config: Hermes.RecursivePartial<Hermes.Config> = useMemo(
    () => ({
      filters: hermesCreatedFilters,
      hooks: {
        onFilterChange: (filters: Hermes.Filters) => {
          // TODO: references are not changing, will need to address this in hermes.
          setHermesCreatedFilters({ ...filters });
        },
        onReset: () => setHermesCreatedFilters({}),
      },
      style: {
        axes: { label: { placement: 'after' } },
        data: {
          series: focusedTrial?.id
            ? new Array(chartData?.trialIds.length).fill(undefined).map((_, index) => ({
                lineWidth: chartData?.trialIds.indexOf(focusedTrial.id) === index ? 3 : 1,
                strokeStyle:
                  chartData?.trialIds.indexOf(focusedTrial.id) === index
                    ? ui.theme.ixOnActive
                    : rgba2str({ ...str2rgba(ui.theme.ixOn), a: 0.1 }),
              }))
            : undefined,
          targetColorScale: colorScale.map((scale) => scale.color),
          targetDimensionKey: selectedMetric ? metricToStr(selectedMetric) : '',
        },
        dimension: { label: { angle: Math.PI / 4, truncate: 24 } },
        padding: [4, 120, 4, 16],
      },
    }),
    [
      hermesCreatedFilters,
      colorScale,
      selectedMetric,
      focusedTrial?.id,
      chartData?.trialIds,
      ui.theme.ixOnActive,
      ui.theme.ixOn,
    ],
  );

  const dimensions = useMemo(() => {
    const newDimensions: Hermes.Dimension[] = selectedHParams.map((key) => {
      const hp = hyperparameters[key] || {};

      if (hp.type === HyperparameterType.Categorical || hp.vals) {
        return {
          categories: hp.vals?.map((val) => (isPrimitive(val) ? val : JSON.stringify(val))) ?? [],
          key,
          label: key,
          type: DimensionType.Categorical,
        };
      } else if (hp.type === HyperparameterType.Log) {
        return { key, label: key, logBase: hp.base, type: DimensionType.Logarithmic };
      }

      return { key, label: key, type: DimensionType.Linear };
    });

    // Add metric as column to parcoords dimension list
    if (chartData?.metricRange && selectedMetric) {
      const key = metricToStr(selectedMetric);
      newDimensions.push(
        selectedScale === Scale.Log
          ? {
              key,
              label: key,
              logBase: 10,
              type: DimensionType.Logarithmic,
            }
          : {
              key,
              label: key,
              type: DimensionType.Linear,
            },
      );
    }

    return newDimensions;
  }, [chartData?.metricRange, hyperparameters, selectedMetric, selectedScale, selectedHParams]);

  const clearSelected = useCallback(() => setSelectedRowKeys([]), []);

  useEffect(() => {
    if (ui.isPageHidden || !selectedMetric) return;

    const canceler = new AbortController();
    const trialMetricsMap: Record<number, number> = {};
    const trialHpTableMap: Record<number, TrialHParams> = {};
    const trialHpMap: Record<string, Record<number, Primitive>> = {};

    setHasLoaded(false);

    readStream<V1TrialsSnapshotResponse>(
      detApi.StreamingInternal.trialsSnapshot(
        experiment.id,
        selectedMetric.name,
        selectedBatch,
        metricTypeParamMap[selectedMetric.type],
        undefined, // custom metric type
        selectedBatchMargin,
        undefined,
        { signal: canceler.signal },
      ),
      (event) => {
        if (!event?.trials || !Array.isArray(event.trials)) return;

        const data: Record<string, Primitive[]> = {};
        let trialMetricRange: Range<number> = defaultNumericRange(true);

        event.trials.forEach((trial) => {
          const id = trial.trialId;
          trialMetricsMap[id] = trial.metric;
          trialMetricRange = updateRange<number>(trialMetricRange, trial.metric);

          // This allows for both typical nested hyperparameters and nested categorgical
          // hyperparameter values to be shown, with HpTrialTable deciding which are displayed.
          const flatHParams = { ...trial.hparams, ...flattenObject(trial.hparams || {}) };

          Object.keys(flatHParams).forEach((hpKey) => {
            const hpValue = flatHParams[hpKey];
            trialHpMap[hpKey] = trialHpMap[hpKey] || {};
            trialHpMap[hpKey][id] = isPrimitive(hpValue) ? hpValue : JSON.stringify(hpValue);
          });

          trialHpTableMap[id] = {
            hparams: clone(flatHParams),
            id,
            metric: trial.metric,
          };
        });

        const trialIds = Object.keys(trialMetricsMap)
          .map((id) => parseInt(id))
          .sort(numericSorter);

        Object.keys(trialHpMap).forEach((hpKey) => {
          data[hpKey] = trialIds.map((trialId) => trialHpMap[hpKey][trialId]);
        });

        // Add metric of interest.
        const metricKey = metricToStr(selectedMetric);
        const metricValues = trialIds.map((id) => trialMetricsMap[id]);
        data[metricKey] = metricValues;

        // Normalize metrics values for parallel coordinates colors.
        const metricRange = getNumericRange(metricValues);

        // Gather hparams for trial table.
        const newTrialHps = trialIds.map((id) => trialHpTableMap[id]);
        setTrialHps(newTrialHps);

        setChartData({
          data,
          metricRange,
          metricValues,
          trialIds,
        });
        setHasLoaded(true);
      },
      (e) => {
        setPageError(e);
        setHasLoaded(true);
      },
    );

    return () => canceler.abort();
  }, [
    experiment.id,
    selectedBatch,
    selectedBatchMargin,
    selectedMetric,
    selectedScale,
    ui.isPageHidden,
  ]);

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

  const handleTrialUnselect = useCallback((trialId: number) => {
    setSelectedRowKeys((rowKeys) => rowKeys.filter((id) => id !== trialId));
  }, []);

  // Reset filtered trial ids when HP Viz filters changes.
  useEffect(() => {
    setFilteredTrialIdMap(undefined);
  }, [selectedBatch, selectedBatchMargin, selectedHParams, selectedMetric]);

  if (pageError) {
    return <Empty description={pageError.message} />;
  } else if (hasLoaded && !chartData) {
    return isExperimentTerminal ? (
      <Empty description="No data to plot." />
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
            <ParallelCoordinates
              config={config}
              data={chartData?.data ?? {}}
              dimensions={dimensions}
              disableInteraction={!!focusedTrial}
            />
          </div>
          {!focusedTrial && !!selectedMetric && (
            <div>
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
                colorScale={colorScale}
                experimentId={experiment.id}
                filteredTrialIdMap={filteredTrialIdMap}
                handleTableRowSelect={handleTableRowSelect}
                hyperparameters={hyperparameters}
                metric={selectedMetric}
                selectedRowKeys={selectedRowKeys}
                selection={true}
                trialHps={trialHps}
              />
            </div>
          )}
          <div className={css.tooltip} ref={tooltipRef}>
            <div className={css.box}>
              <div className={css.row}>
                <div>Trial Id:</div>
                <div ref={trialIdRef} />
              </div>
              <div className={css.row}>
                <div>Metric:</div>
                <div ref={metricValueRef} />
              </div>
            </div>
          </div>
        </div>
      </Section>
      {showCompareTrials && (
        <TrialsComparisonModal
          experiment={experiment}
          trialIds={selectedRowKeys}
          visible={showCompareTrials}
          onCancel={() => setShowCompareTrials(false)}
          onUnselect={handleTrialUnselect}
        />
      )}
    </div>
  );
};

export default HpParallelCoordinates;
