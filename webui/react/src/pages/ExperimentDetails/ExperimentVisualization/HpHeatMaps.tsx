import { Alert } from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';

import ColorLegend from 'components/ColorLegend';
import Grid, { GridMode } from 'components/Grid';
import { GridListView } from 'components/GridListRadioGroup';
import Message, { MessageType } from 'components/Message';
import MetricBadgeTag from 'components/MetricBadgeTag';
import ScatterPlot from 'components/ScatterPlot';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import useResize from 'hooks/useResize';
import { V1TrialsSnapshotResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import {
  ExperimentBase, ExperimentHyperParamType, MetricName, MetricType, metricTypeParamMap, Range,
} from 'types';
import { getColorScale } from 'utils/chart';
import { isNumber, isObject } from 'utils/data';
import { metricNameToStr } from 'utils/string';
import { terminalRunStates } from 'utils/types';

import css from './HpHeatMaps.module.scss';

interface Props {
  experiment: ExperimentBase;
  hParams: string[];
  filters?: React.ReactNode;
  selectedBatch: number;
  selectedBatchMargin: number;
  selectedHParams: string[];
  selectedMetric: MetricName;
  selectedView: GridListView;
}

interface HpData {
  hpLogScales: Record<string, boolean>;
  hpMetrics: Record<string, number[]>;
  hpValues: Record<string, number[]>;
  metricRange: Range<number>;
  trialIds: number[];
}

const generateHpKey = (hParam1: string, hParam2: string): string => {
  return `${hParam1}:${hParam2}`;
};

const HpHeatMaps: React.FC<Props> = ({
  experiment,
  hParams,
  filters,
  selectedBatch,
  selectedBatchMargin,
  selectedHParams,
  selectedMetric,
  selectedView,
}: Props) => {
  const baseRef = useRef<HTMLDivElement>(null);
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ chartData, setChartData ] = useState<HpData>();
  const [ pageError, setPageError ] = useState<Error>();
  const resize = useResize(baseRef);

  const isExperimentTerminal = terminalRunStates.has(experiment.state);
  const isListView = selectedView === GridListView.List;

  const smallerIsBetter = useMemo(() => {
    if (selectedMetric.type === MetricType.Validation &&
        selectedMetric.name === experiment.config.searcher.metric) {
      return experiment.config.searcher.smallerIsBetter;
    }
    return undefined;
  }, [ experiment.config.searcher, selectedMetric ]);

  const colorScale = useMemo(() => {
    return getColorScale(chartData?.metricRange, smallerIsBetter);
  }, [ chartData, smallerIsBetter ]);

  useEffect(() => {
    const canceler = new AbortController();
    const trialIds: number[] = [];
    const hpMetricMap: Record<number, Record<string, number>> = {};
    const hpValueMap: Record<number, Record<string, number>> = {};

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

        const hpLogScaleMap: Record<string, boolean> = {};
        const hpMetrics: Record<string, number[]> = {};
        const hpValues: Record<string, number[]> = {};
        const metricRange: Range<number> = [ Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY ];

        event.trials.forEach(trial => {
          if (!isObject(trial.hparams)) return;

          const trialId = trial.trialId;
          const trialHParams = Object.keys(trial.hparams)
            .filter(hParam => isNumber(trial.hparams[hParam]))
            .filter(hParam => hParams.includes(hParam))
            .sort((a, b) => a.localeCompare(b, 'en'));

          trialIds.push(trialId);
          hpMetricMap[trialId] = hpMetricMap[trialId] || {};
          hpValueMap[trialId] = hpValueMap[trialId] || {};
          trialHParams.forEach(hParam1 => {
            hpValueMap[trialId][hParam1] = trial.hparams[hParam1];
            trialHParams.forEach(hParam2 => {
              const key = generateHpKey(hParam1, hParam2);
              hpMetricMap[trialId][key] = trial.metric;
            });
          });

          if (trial.metric < metricRange[0]) metricRange[0] = trial.metric;
          if (trial.metric > metricRange[1]) metricRange[1] = trial.metric;
        });

        hParams.forEach(hParam1 => {
          const hp = (experiment.config.hyperparameters || {})[hParam1];
          if (hp.type === ExperimentHyperParamType.Log) hpLogScaleMap[hParam1] = true;

          hpValues[hParam1] = trialIds.map(trialId => hpValueMap[trialId][hParam1]);
          hParams.forEach(hParam2 => {
            const key = generateHpKey(hParam1, hParam2);
            hpMetrics[key] = trialIds.map(trialId => hpMetricMap[trialId][key]);
          });
        });

        setChartData({
          hpLogScales: hpLogScaleMap,
          hpMetrics,
          hpValues,
          metricRange,
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
        <>
          <div className={css.legend}>
            <ColorLegend
              colorScale={colorScale}
              title={<MetricBadgeTag metric={selectedMetric} />} />
          </div>
          <div className={css.charts}>
            <Grid
              border={true}
              minItemWidth={resize.width > 320 ? 35 : 27}
              mode={!isListView ? selectedHParams.length : GridMode.AutoFill}>
              {selectedHParams.map(hParam1 => selectedHParams.map(hParam2 => {
                const key = generateHpKey(hParam1, hParam2);
                return <ScatterPlot
                  colorScale={colorScale}
                  height={350}
                  key={key}
                  valueLabel={metricNameToStr(selectedMetric)}
                  values={chartData.hpMetrics[key]}
                  width={350}
                  x={chartData.hpValues[hParam1]}
                  xLabel={hParam1}
                  xLogScale={chartData.hpLogScales[hParam1]}
                  y={chartData.hpValues[hParam2]}
                  yLabel={hParam2}
                  yLogScale={chartData.hpLogScales[hParam2]}
                />;
              }))}
            </Grid>
          </div>
        </>
      );
    }
  }

  return (
    <div className={css.base} ref={baseRef}>
      <Section bodyBorder filters={filters} id="hp-visualization" title="HP Heat Maps">
        <div className={css.container}>{content}</div>
      </Section>
    </div>
  );
};

export default HpHeatMaps;
