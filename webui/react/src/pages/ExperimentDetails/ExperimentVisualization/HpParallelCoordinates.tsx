import { Alert } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Message, { MessageType } from 'components/Message';
import ParallelCoordinates, {
  Constraint, Dimension, DimensionType, dimensionTypeMap,
} from 'components/ParallelCoordinates';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import TableBatch from 'components/TableBatch';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { openOrCreateTensorboard } from 'services/api';
import { V1TrialsSnapshotResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import {
  ExperimentAction as Action, CommandTask, ExperimentBase, Hyperparameter,
  HyperparameterType, MetricName, MetricType, metricTypeParamMap, Primitive, Range,
} from 'types';
import { defaultNumericRange, getColorScale, getNumericRange, updateRange } from 'utils/chart';
import { clone, flattenObject } from 'utils/data';
import { numericSorter } from 'utils/sort';
import { metricNameToStr } from 'utils/string';
import { terminalRunStates } from 'utils/types';
import { openCommand } from 'wait';

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
  selectedMetric: MetricName;
}

interface HpTrialData {
  data: Record<string, Primitive[]>;
  metricRange?: Range<number>;
  metricValues: number[];
  trialIds: number[];
}

const HpParallelCoordinates: React.FC<Props> = ({
  experiment,
  filters,
  fullHParams,
  selectedBatch,
  selectedBatchMargin,
  selectedHParams,
  selectedMetric,
}: Props) => {
  const tooltipRef = useRef<HTMLDivElement>(null);
  const trialIdRef = useRef<HTMLDivElement>(null);
  const metricValueRef = useRef<HTMLDivElement>(null);
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ chartData, setChartData ] = useState<HpTrialData>();
  const [ trialHps, setTrialHps ] = useState<TrialHParams[]>([]);
  const [ pageError, setPageError ] = useState<Error>();
  const [ filteredTrialIdMap, setFilteredTrialIdMap ] = useState<Record<number, boolean>>();
  const [ selectedRowKeys, setSelectedRowKeys ] = useState<number[]>([]);
  const [ showCompareTrials, setShowCompareTrials ] = useState(false);

  const hyperparameters = useMemo(() => {
    return fullHParams.reduce((acc, key) => {
      acc[key] = experiment.hyperparameters[key];
      return acc;
    }, {} as Record<string, Hyperparameter>);
  }, [ experiment.hyperparameters, fullHParams ]);

  const isExperimentTerminal = terminalRunStates.has(experiment.state);

  const smallerIsBetter = useMemo(() => {
    if (selectedMetric.type === MetricType.Validation &&
        selectedMetric.name === experiment.config.searcher.metric) {
      return experiment.config.searcher.smallerIsBetter;
    }
    return undefined;
  }, [ experiment.config.searcher, selectedMetric ]);

  const colorScale = useMemo(() => {
    return getColorScale(chartData?.metricRange, smallerIsBetter);
  }, [ chartData?.metricRange, smallerIsBetter ]);

  const dimensions = useMemo(() => {
    const newDimensions = selectedHParams.map(key => {
      const hp = hyperparameters[key] || {};
      const dimension: Dimension = {
        label: key,
        type: hp.type ? dimensionTypeMap[hp.type] : DimensionType.Scalar,
      };

      if (hp.vals) dimension.categories = hp.vals;
      if (hp.minval != null && hp.maxval != null) {
        const isLogarithmic = hp.type === HyperparameterType.Log;
        dimension.range = isLogarithmic ?
          [ 10 ** hp.minval, 10 ** hp.maxval ] : [ hp.minval, hp.maxval ];
      }

      return dimension;
    });

    // Add metric as column to parcoords dimension list
    if (chartData?.metricRange) {
      newDimensions.push({
        label: metricNameToStr(selectedMetric),
        range: chartData.metricRange,
        type: DimensionType.Scalar,
      });
    }

    return newDimensions;
  }, [ chartData?.metricRange, hyperparameters, selectedMetric, selectedHParams ]);

  const handleChartFilter = useCallback((constraints: Record<string, Constraint>) => {
    if (!chartData) return;

    // Figure out which trials fit within the user provided constraints.
    const newFilteredTrialIdMap = chartData.trialIds.reduce((acc, trialId) => {
      acc[trialId] = true;
      return acc;
    }, {} as Record<number, boolean>);

    Object.entries(constraints).forEach(([ key, constraint ]) => {
      if (!constraint) return;
      if (!chartData.data[key]) return;

      const range = constraint.range;
      const values = chartData.data[key];
      values.forEach((value, index) => {
        if (constraint.values && constraint.values.includes(value)) return;
        if (!constraint.values && value >= range[0] && value <= range[1]) return;
        const trialId = chartData.trialIds[index];
        newFilteredTrialIdMap[trialId] = false;
      });
    });

    setFilteredTrialIdMap(newFilteredTrialIdMap);
  }, [ chartData ]);

  const clearSelected = useCallback(() => {
    setSelectedRowKeys([]);
  }, []);

  useEffect(() => {
    const canceler = new AbortController();
    const trialMetricsMap: Record<number, number> = {};
    const trialHpTableMap: Record<number, TrialHParams> = {};
    const trialHpMap: Record<string, Record<number, Primitive>> = {};

    setHasLoaded(false);

    consumeStream<V1TrialsSnapshotResponse>(
      detApi.StreamingInternal.determinedTrialsSnapshot(
        experiment.id,
        selectedMetric.name,
        metricTypeParamMap[selectedMetric.type],
        selectedBatch,
        selectedBatchMargin,
        undefined,
        { signal: canceler.signal },
      ),
      event => {
        if (!event || !event.trials || !Array.isArray(event.trials)) return;

        const data: Record<string, Primitive[]> = {};
        let trialMetricRange: Range<number> = defaultNumericRange(true);

        event.trials.forEach(trial => {
          const id = trial.trialId;
          trialMetricsMap[id] = trial.metric;
          trialMetricRange = updateRange<number>(trialMetricRange, trial.metric);

          const flatHParams = flattenObject(trial.hparams || {});
          Object.keys(flatHParams).forEach(hpKey => {
            const hpValue = flatHParams[hpKey];
            trialHpMap[hpKey] = trialHpMap[hpKey] || {};
            trialHpMap[hpKey][id] = hpValue;
          });

          trialHpTableMap[id] = {
            hparams: clone(flatHParams),
            id,
            metric: trial.metric,
          };
        });

        const trialIds = Object.keys(trialMetricsMap)
          .map(id => parseInt(id))
          .sort(numericSorter);

        Object.keys(trialHpMap).forEach(hpKey => {
          data[hpKey] = trialIds.map(trialId => trialHpMap[hpKey][trialId]);
        });

        // Add metric of interest.
        const metricKey = metricNameToStr(selectedMetric);
        const metricValues = trialIds.map(id => trialMetricsMap[id]);
        data[metricKey] = metricValues;

        // Normalize metrics values for parallel coordinates colors.
        const metricRange = getNumericRange(metricValues);

        // Gather hparams for trial table.
        const newTrialHps = trialIds.map(id => trialHpTableMap[id]);
        setTrialHps(newTrialHps);

        setChartData({
          data,
          metricRange,
          metricValues,
          trialIds,
        });
        setHasLoaded(true);
      },
    ).catch(e => {
      setPageError(e);
      setHasLoaded(true);
    });

    return () => canceler.abort();
  }, [ experiment.id, selectedBatch, selectedBatchMargin, selectedMetric ]);

  const sendBatchActions = useCallback(async (action: Action) => {
    if (action === Action.OpenTensorBoard) {
      return await openOrCreateTensorboard({ trialIds: selectedRowKeys });
    } else if (action === Action.CompareTrials) {
      return setShowCompareTrials(true);
    }
  }, [ selectedRowKeys ]);

  const submitBatchAction = useCallback(async (action: Action) => {
    try {
      const result = await sendBatchActions(action);
      if (action === Action.OpenTensorBoard && result) {
        openCommand(result as CommandTask);
      }
    } catch (e) {
      const publicSubject = action === Action.OpenTensorBoard ?
        'Unable to View TensorBoard for Selected Trials' :
        `Unable to ${action} Selected Trials`;
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: 'Please try again later.',
        publicSubject,
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ sendBatchActions ]);

  const handleTableRowSelect = useCallback(rowKeys => setSelectedRowKeys(rowKeys), []);

  const handleTrialUnselect = useCallback((trialId: number) =>
    setSelectedRowKeys(rowKeys => rowKeys.filter(id => id !== trialId)), []);

  if (pageError) {
    return <Message title={pageError.message} />;
  } else if (hasLoaded && !chartData) {
    return isExperimentTerminal ? (
      <Message title="No data to plot." type={MessageType.Empty} />
    ) : (
      <div className={css.waiting}>
        <Alert
          description="Please wait until the experiment is further along."
          message="Not enough data points to plot." />
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
              colorScale={colorScale}
              colorScaleKey={metricNameToStr(selectedMetric)}
              data={chartData?.data || {}}
              dimensions={dimensions}
              onFilter={handleChartFilter}
            />
          </div>
          <div>
            <TableBatch
              actions={[
                { label: Action.OpenTensorBoard, value: Action.OpenTensorBoard },
                { label: Action.CompareTrials, value: Action.CompareTrials },
              ]}
              selectedRowCount={selectedRowKeys.length}
              onAction={action => submitBatchAction(action as Action)}
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
              trialIds={chartData?.trialIds || []}
            />
          </div>
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
      {showCompareTrials &&
      <TrialsComparisonModal
        experiment={experiment}
        trials={selectedRowKeys}
        visible={showCompareTrials}
        onCancel={() => setShowCompareTrials(false)}
        onUnselect={handleTrialUnselect} />}
    </div>
  );
};

export default HpParallelCoordinates;
