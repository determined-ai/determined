import { Alert } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import LearningCurveChart from 'components/LearningCurveChart';
import Message, { MessageType } from 'components/Message';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import { isNewTabClickEvent, openBlank, paths, routeAll } from 'routes/utils';
import { V1TrialsSampleResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import {
  ExperimentBase, ExperimentHyperParam, MetricName, metricTypeParamMap, RunState,
} from 'types';
import { terminalRunStates } from 'utils/types';

import HpTrialTable, { TrialHParams } from './HpTrialTable';
import css from './LearningCurve.module.scss';

interface Props {
  experiment: ExperimentBase;
  filters?: React.ReactNode;
  fullHParams: string[];
  selectedMaxTrial: number;
  selectedMetric: MetricName
}

const MAX_DATAPOINTS = 5000;

const LearningCurve: React.FC<Props> = ({
  experiment,
  filters,
  fullHParams,
  selectedMaxTrial,
  selectedMetric,
}: Props) => {
  const [ trialIds, setTrialIds ] = useState<number[]>([]);
  const [ batches, setBatches ] = useState<number[]>([]);
  const [ chartData, setChartData ] = useState<(number | null)[][]>([]);
  const [ trialHps, setTrialHps ] = useState<TrialHParams[]>([]);
  const [ highlightedTrialId, setHighlightedTrialId ] = useState<number>();
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ pageError, setPageError ] = useState<Error>();

  const hasTrials = trialHps.length !== 0;
  const isExperimentTerminal = terminalRunStates.has(experiment.state as RunState);

  const hyperparameters = useMemo(() => {
    return fullHParams.reduce((acc, key) => {
      acc[key] = experiment.config.hyperparameters[key];
      return acc;
    }, {} as Record<string, ExperimentHyperParam>);
  }, [ experiment.config.hyperparameters, fullHParams ]);

  const handleTrialClick = useCallback((event: MouseEvent, trialId: number) => {
    const href = paths.trialDetails(trialId, experiment.id);
    if (isNewTabClickEvent(event)) openBlank(href);
    else routeAll(href);
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

  useEffect(() => {
    const canceler = new AbortController();
    const trialIdsMap: Record<number, number> = {};
    const trialDataMap: Record<number, number[]> = {};
    const trialHpMap: Record<number, TrialHParams> = {};
    const batchesMap: Record<number, number> = {};
    const metricsMap: Record<number, Record<number, number>> = {};

    setHasLoaded(false);

    consumeStream<V1TrialsSampleResponse>(
      detApi.StreamingInternal.determinedTrialsSample(
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
      event => {
        if (!event || !event.trials || !Array.isArray(event.trials)) return;

        /*
         * Cache trial ids, hparams, batches and metric values into easily searchable
         * dictionaries, then construct the necessary data structures to render the
         * chart and the table.
         */

        (event.promotedTrials || []).forEach(trialId => trialIdsMap[trialId] = trialId);
        (event.demotedTrials || []).forEach(trialId => delete trialIdsMap[trialId]);
        const newTrialIds = Object.values(trialIdsMap);
        setTrialIds(newTrialIds);

        (event.trials || []).forEach(trial => {
          const id = trial.trialId;
          const hasHParams = Object.keys(trial.hparams || {}).length !== 0;

          if (hasHParams && !trialHpMap[id]) {
            trialHpMap[id] = { hparams: trial.hparams, id, metric: null };
          }

          trialDataMap[id] = trialDataMap[id] || [];
          metricsMap[id] = metricsMap[id] || {};

          trial.data.forEach(datapoint => {
            batchesMap[datapoint.batches] = datapoint.batches;
            metricsMap[id][datapoint.batches] = datapoint.value;
            trialHpMap[id].metric = datapoint.value;
          });
        });

        const newTrialHps = newTrialIds.map(id => trialHpMap[id]);
        setTrialHps(newTrialHps);

        const newBatches = Object.values(batchesMap);
        setBatches(newBatches);

        const newChartData = newTrialIds.map(trialId => newBatches.map(batch => {
          const value = metricsMap[trialId][batch];
          return value != null ? value : null;
        }));
        setChartData(newChartData);

        // One successful event as come through.
        setHasLoaded(true);
      },
    ).catch(e => {
      setPageError(e);
      setHasLoaded(true);
    });

    return () => canceler.abort();
  }, [ experiment.id, selectedMaxTrial, selectedMetric ]);

  if (pageError) {
    return <Message title={pageError.message} />;
  } else if (hasLoaded && !hasTrials) {
    return isExperimentTerminal ? (
      <Message title="No learning curve data to show." type={MessageType.Empty} />
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
            <LearningCurveChart
              data={chartData}
              focusedTrialId={highlightedTrialId}
              selectedMetric={selectedMetric}
              trialIds={trialIds}
              xValues={batches}
              onTrialClick={handleTrialClick}
              onTrialFocus={handleTrialFocus} />
          </div>
          <HpTrialTable
            experimentId={experiment.id}
            highlightedTrialId={highlightedTrialId}
            hyperparameters={hyperparameters}
            metric={selectedMetric}
            trialHps={trialHps}
            trialIds={trialIds}
            onMouseEnter={handleTableMouseEnter}
            onMouseLeave={handleTableMouseLeave}
          />
        </div>
      </Section>
    </div>
  );
};

export default LearningCurve;
