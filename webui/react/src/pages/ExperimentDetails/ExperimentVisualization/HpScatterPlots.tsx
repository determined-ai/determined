import { Alert } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import GalleryModal from 'components/GalleryModal';
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
import { metricNameToStr } from 'utils/string';
import { terminalRunStates } from 'utils/types';

import css from './HpScatterPlots.module.scss';

interface Props {
  experiment: ExperimentBase;
  filters?: React.ReactNode;
  fullHParams: string[];
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
  filters,
  fullHParams,
  selectedBatch,
  selectedBatchMargin,
  selectedHParams,
  selectedMetric,
}: Props) => {
  const baseRef = useRef<HTMLDivElement>(null);
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ chartData, setChartData ] = useState<HpMetricData>();
  const [ pageError, setPageError ] = useState<Error>();
  const [ activeHParam, setActiveHParam ] = useState<string>();
  const [ galleryHeight, setGalleryHeight ] = useState<number>(450);

  const resize = useResize(baseRef);
  const isExperimentTerminal = terminalRunStates.has(experiment.state);

  const handleChartClick = useCallback((hParam: string) => setActiveHParam(hParam), []);

  const handleGalleryClose = useCallback(() => setActiveHParam(undefined), []);

  const handleGalleryNext = useCallback(() => {
    setActiveHParam(prev => {
      if (!prev) return prev;
      const index = selectedHParams.indexOf(prev);
      if (index === -1) return prev;
      const nextIndex = index === selectedHParams.length - 1 ? 0 : index + 1;
      return selectedHParams[nextIndex];
    });
  }, [ selectedHParams ]);

  const handleGalleryPrevious = useCallback(() => {
    setActiveHParam(prev => {
      if (!prev) return prev;
      const index = selectedHParams.indexOf(prev);
      if (index === -1) return prev;
      const prevIndex = index === 0 ? selectedHParams.length - 1 : index - 1;
      return selectedHParams[prevIndex];
    });
  }, [ selectedHParams ]);

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

          fullHParams.forEach(hParam => {
            hpTrialMap[hParam] = hpTrialMap[hParam] || {};
            hpTrialMap[hParam][trialId] = hpTrialMap[hParam][trialId] || {};
            hpTrialMap[hParam][trialId] = {
              hp: trial.hparams[hParam],
              metric: trial.metric,
            };
          });
        });

        fullHParams.forEach(hParam => {
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
  }, [ experiment, fullHParams, selectedBatch, selectedBatchMargin, selectedMetric ]);

  useEffect(() => setGalleryHeight(resize.height), [ resize ]);

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
    <div className={css.base} ref={baseRef}>
      <Section bodyBorder bodyNoPadding bodyScroll filters={filters} loading={!hasLoaded}>
        <div className={css.container}>
          {chartData?.trialIds.length === 0 ? (
            <Message title="No data to plot." type={MessageType.Empty} />
          ) : (
            <Grid
              border={true}
              minItemWidth={resize.width > 320 ? 35 : 27}
              mode={GridMode.AutoFill}>
              {selectedHParams.map(hParam => (
                <div key={hParam} onClick={() => handleChartClick(hParam)}>
                  <ScatterPlot
                    disableZoom
                    x={chartData?.hpValues[hParam] || []}
                    xLabel={hParam}
                    xLogScale={chartData?.hpLogScales[hParam]}
                    y={chartData?.metricValues[hParam] || []}
                    yLabel={metricNameToStr(selectedMetric)}
                  />
                </div>
              ))}
            </Grid>
          )}
        </div>
      </Section>
      <GalleryModal
        height={galleryHeight}
        visible={!!activeHParam}
        onCancel={handleGalleryClose}
        onNext={handleGalleryNext}
        onPrevious={handleGalleryPrevious}>
        {activeHParam && <ScatterPlot
          height={galleryHeight}
          x={chartData?.hpValues[activeHParam] || []}
          xLabel={activeHParam}
          xLogScale={chartData?.hpLogScales[activeHParam]}
          y={chartData?.metricValues[activeHParam] || []}
          yLabel={metricNameToStr(selectedMetric)}
        />}
      </GalleryModal>
    </div>
  );
};

export default ScatterPlots;
