import { Alert } from 'antd';
import React, { useEffect, useRef, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import Message, { MessageType } from 'components/Message';
import ScatterPlot from 'components/ScatterPlot';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import useResize from 'hooks/useResize';
import { V1TrialsSnapshotResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import { ExperimentBase, ExperimentHyperParamType, MetricName, metricTypeParamMap } from 'types';
import { isNumber } from 'utils/data';
import { metricNameToStr } from 'utils/string';
import { terminalRunStates } from 'utils/types';

import css from './HpScatterPlots.module.scss';

interface Props {
  experiment: ExperimentBase;
  filters?: React.ReactNode;
  hParams: string[];
  selectedBatch: number;
  selectedBatchMargin: number;
  selectedHParams: string[];
  selectedMetric: MetricName;
}

interface HpMetricData {
  hpLogScales: Record<string, boolean>;
  hpValues: Record<string, number[]>;
  metricValues: Record<string, number[]>;
  trialIds: number[];
}

const ScatterPlots: React.FC<Props> = ({
  experiment,
  hParams,
  filters,
  selectedBatch,
  selectedBatchMargin,
  selectedHParams,
  selectedMetric,
}: Props) => {
  const baseRef = useRef<HTMLDivElement>(null);
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ chartData, setChartData ] = useState<HpMetricData>();
  const [ pageError, setPageError ] = useState<Error>();

  const resize = useResize(baseRef);
  const isExperimentTerminal = terminalRunStates.has(experiment.state);

  useEffect(() => {
    const canceler = new AbortController();
    const trialIds: number[] = [];
    const hpTrialMap: Record<string, Record<number, { hp: number, metric: number }>> = {};

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

        const hpMetricMap: Record<string, number[]> = {};
        const hpValueMap: Record<string, number[]> = {};
        const hpLogScaleMap: Record<string, boolean> = {};

        event.trials.forEach(trial => {
          const trialId = trial.trialId;
          trialIds.push(trialId);

          hParams.forEach(hParam => {
            if (!isNumber(trial.hparams[hParam])) return;

            hpTrialMap[hParam] = hpTrialMap[hParam] || {};
            hpTrialMap[hParam][trialId] = hpTrialMap[hParam][trialId] || {};
            hpTrialMap[hParam][trialId] = {
              hp: trial.hparams[hParam],
              metric: trial.metric,
            };
          });
        });

        hParams.forEach(hParam => {
          const hp = (experiment.config.hyperparameters || {})[hParam];
          if (hp.type === ExperimentHyperParamType.Log) hpLogScaleMap[hParam] = true;

          hpMetricMap[hParam] = [];
          hpValueMap[hParam] = [];
          trialIds.forEach(trialId => {
            const map = (hpTrialMap[hParam] || {})[trialId] || {};
            hpMetricMap[hParam].push(map.metric);
            hpValueMap[hParam].push(map.hp);
          });
        });

        setChartData({
          hpLogScales: hpLogScaleMap,
          hpValues: hpValueMap,
          metricValues: hpMetricMap,
          trialIds,
        });
        setHasLoaded(true);
      },
    ).catch(e => {
      setPageError(e);
      setHasLoaded(true);
    });

    return () => canceler.abort();
  }, [ experiment, hParams, selectedBatch, selectedBatchMargin, selectedMetric ]);

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

  let content = <Spinner />;
  if (hasLoaded && chartData) {
    if (chartData.trialIds.length === 0) {
      content = <Message title="No data to plot." type={MessageType.Empty} />;
    } else {
      content = (
        <Grid
          border={true}
          minItemWidth={resize.width > 320 ? 35 : 27}
          mode={GridMode.AutoFill}>
          {selectedHParams.map(hParam => (
            <ScatterPlot
              key={hParam}
              x={chartData.hpValues[hParam]}
              xLabel={hParam}
              xLogScale={chartData.hpLogScales[hParam]}
              y={chartData.metricValues[hParam]}
              yLabel={metricNameToStr(selectedMetric)} />
          ))}
        </Grid>
      );
    }
  }

  return (
    <div className={css.base} ref={baseRef}>
      <Section
        bodyBorder
        bodyScroll
        filters={filters}
        id="hp-visualization"
        title="HP Scatter Plots">
        <div className={css.container}>{content}</div>
      </Section>
    </div>
  );
};

export default ScatterPlots;
