import { Alert } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import LearningCurveChart from 'components/LearningCurveChart';
import Section from 'components/Section';
import TableBatch from 'components/TableBatch';
import { terminalRunStates } from 'constants/states';
import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';
import { openOrCreateTensorBoard } from 'services/api';
import { V1TrialsSampleResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner/Spinner';
import { flattenObject } from 'shared/utils/data';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { isNewTabClickEvent, openBlank, routeToReactUrl } from 'shared/utils/routes';
import {
  ExperimentAction as Action, CommandTask, ExperimentBase, Hyperparameter, MetricName,
  metricTypeParamMap, RunState, Scale,
} from 'types';
import handleError from 'utils/error';
import { openCommand } from 'utils/wait';

import TrialsComparisonModal from '../TrialsComparisonModal';

import HpTrialTable, { TrialHParams } from './HpTrialTable';
import css from './LearningCurve.module.scss';

interface Props {
  experiment: ExperimentBase;
  filters?: React.ReactNode;
  fullHParams: string[];
  selectedMaxTrial: number;
  selectedMetric: MetricName
  selectedScale: Scale;
}

const MAX_DATAPOINTS = 5000;

const LearningCurve: React.FC<Props> = ({
  experiment,
  filters,
  fullHParams,
  selectedMaxTrial,
  selectedMetric,
  selectedScale,
}: Props) => {
  const { ui } = useStore();
  const [ trialIds, setTrialIds ] = useState<number[]>([]);
  const [ batches, setBatches ] = useState<number[]>([]);
  const [ chartData, setChartData ] = useState<(number | null)[][]>([]);
  const [ trialHps, setTrialHps ] = useState<TrialHParams[]>([]);
  const [ highlightedTrialId, setHighlightedTrialId ] = useState<number>();
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ pageError, setPageError ] = useState<Error>();
  const [ selectedRowKeys, setSelectedRowKeys ] = useState<number[]>([]);
  const [ showCompareTrials, setShowCompareTrials ] = useState(false);

  const hasTrials = trialHps.length !== 0;
  const isExperimentTerminal = terminalRunStates.has(experiment.state as RunState);

  const hyperparameters = useMemo(() => {
    return fullHParams.reduce((acc, key) => {
      acc[key] = experiment.hyperparameters[key];
      return acc;
    }, {} as Record<string, Hyperparameter>);
  }, [ experiment.hyperparameters, fullHParams ]);

  const handleTrialClick = useCallback((event: MouseEvent, trialId: number) => {
    const href = paths.trialDetails(trialId, experiment.id);
    if (isNewTabClickEvent(event)) openBlank(href);
    else routeToReactUrl(href);
  }, [ experiment.id ]);

  const handleTrialFocus = useCallback((trialId: number | null) => {
    setHighlightedTrialId(trialId != null ? trialId : undefined);
  }, []);

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
        if (!event || !event.trials || !Array.isArray(event.trials)) return;

        /*
         * Cache trial ids, hparams, batches and metric values into easily searchable
         * dictionaries, then construct the necessary data structures to render the
         * chart and the table.
         */

        (event.promotedTrials || []).forEach((trialId) => trialIdsMap[trialId] = trialId);
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

          trial.data.forEach((datapoint) => {
            batchesMap[datapoint.batches] = datapoint.batches;
            metricsMap[id][datapoint.batches] = datapoint.value;
            trialHpMap[id].metric = datapoint.value;
          });
        });

        const newTrialHps = newTrialIds.map((id) => trialHpMap[id]);
        setTrialHps(newTrialHps);

        const newBatches = Object.values(batchesMap);
        setBatches(newBatches);

        const newChartData = newTrialIds.map((trialId) => newBatches.map((batch) => {
          /**
           * TODO: filtering NaN, +/- Infinity for now, but handle it later with
           * dynamic min/max ranges via uPlot.Scales.
           */
          const value = metricsMap[trialId][batch];
          return Number.isFinite(value) ? value : null;
        }));
        setChartData(newChartData);

        // One successful event as come through.
        setHasLoaded(true);
      },
    ).catch((e) => {
      setPageError(e);
      setHasLoaded(true);
    });

    return () => canceler.abort();
  }, [ experiment.id, selectedMaxTrial, selectedMetric, ui.isPageHidden ]);

  const sendBatchActions = useCallback(async (action: Action) => {
    if (action === Action.OpenTensorBoard) {
      return await openOrCreateTensorBoard({ trialIds: selectedRowKeys });
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
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject,
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ sendBatchActions ]);

  const handleTableRowSelect = useCallback((rowKeys) => setSelectedRowKeys(rowKeys), []);

  const handleTrialUnselect = useCallback((trialId: number) =>
    setSelectedRowKeys((rowKeys) => rowKeys.filter((id) => id !== trialId)), []);

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
            trialIds={trialIds}
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
