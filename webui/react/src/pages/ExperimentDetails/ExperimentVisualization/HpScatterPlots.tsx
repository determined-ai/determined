import { Alert } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import GalleryModal from 'components/GalleryModal';
import Grid, { GridMode } from 'components/Grid';
import Message, { MessageType } from 'components/Message';
import Section from 'components/Section';
import Spinner from 'components/Spinner/Spinner';
import { FacetedData, UPlotScatterProps } from 'components/UPlot/types';
import UPlotScatter from 'components/UPlot/UPlotScatter';
import { terminalRunStates } from 'constants/states';
import useResize from 'hooks/useResize';
import { V1TrialsSnapshotResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import useUI from 'stores/contexts/UI';
import { Primitive } from 'types';
import { ExperimentBase, HyperparameterType, Metric, metricTypeParamMap, Scale } from 'types';
import { flattenObject, isBoolean, isString } from 'utils/data';
import { metricToStr } from 'utils/metric';

import css from './HpScatterPlots.module.scss';

interface Props {
  experiment: ExperimentBase;
  filters?: React.ReactNode;
  fullHParams: string[];
  selectedBatch: number;
  selectedBatchMargin: number;
  selectedHParams: string[];
  selectedMetric?: Metric;
  selectedScale: Scale;
}

interface HpMetricData {
  hpLabels: Record<string, string[]>;
  hpLogScales: Record<string, boolean>;
  hpValues: Record<string, number[]>;
  metricValues: Record<string, (number | undefined)[]>;
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
  selectedScale,
}: Props) => {
  const { ui } = useUI();
  const baseRef = useRef<HTMLDivElement>(null);
  const [hasLoaded, setHasLoaded] = useState(false);
  const [chartData, setChartData] = useState<HpMetricData>();
  const [pageError, setPageError] = useState<Error>();
  const [activeHParam, setActiveHParam] = useState<string>();
  const [galleryHeight, setGalleryHeight] = useState<number>(450);

  const yScaleKey = selectedScale === Scale.Log ? 'yLog' : 'y';

  const resize = useResize(baseRef);
  const isExperimentTerminal = terminalRunStates.has(experiment.state);

  const chartProps = useMemo(() => {
    if (!chartData || !selectedMetric) return undefined;

    return selectedHParams.reduce((acc, hParam) => {
      const xLabel = hParam;
      const yLabel = metricToStr(selectedMetric);
      const title = `${yLabel} (y) vs ${xLabel} (x)`;
      const hpLabels = chartData?.hpLabels[hParam];
      const isLogarithmic = chartData?.hpLogScales[hParam];
      const isCategorical = hpLabels?.length !== 0;
      const xScaleKey = isCategorical ? 'xCategorical' : isLogarithmic ? 'xLog' : 'x';
      const xSplits = isCategorical
        ? new Array(hpLabels.length).fill(0).map((x, i) => i)
        : undefined;
      const xValues = isCategorical ? hpLabels : undefined;
      acc[hParam] = {
        data: [
          null,
          [
            chartData?.hpValues[hParam] || [],
            chartData?.metricValues[hParam] || [],
            null,
            null,
            null,
            chartData?.trialIds || [],
          ],
        ] as FacetedData,
        options: {
          axes: [
            {
              scale: xScaleKey,
              splits: xSplits,
              values: xValues,
            },
            { scale: yScaleKey },
          ],
          cursor: { drag: { setScale: false, x: false, y: false } },
          title,
        },
        tooltipLabels: [xLabel, yLabel, null, null, null, 'trial ID'],
      };
      return acc;
    }, {} as Record<string, UPlotScatterProps>);
  }, [chartData, selectedHParams, selectedMetric, yScaleKey]);

  const handleChartClick = useCallback((hParam: string) => setActiveHParam(hParam), []);

  const handleGalleryClose = useCallback(() => setActiveHParam(undefined), []);

  const handleGalleryNext = useCallback(() => {
    setActiveHParam((prev) => {
      if (!prev) return prev;
      const index = selectedHParams.indexOf(prev);
      if (index === -1) return prev;
      const nextIndex = index === selectedHParams.length - 1 ? 0 : index + 1;
      return selectedHParams[nextIndex];
    });
  }, [selectedHParams]);

  const handleGalleryPrevious = useCallback(() => {
    setActiveHParam((prev) => {
      if (!prev) return prev;
      const index = selectedHParams.indexOf(prev);
      if (index === -1) return prev;
      const prevIndex = index === 0 ? selectedHParams.length - 1 : index - 1;
      return selectedHParams[prevIndex];
    });
  }, [selectedHParams]);

  useEffect(() => {
    if (ui.isPageHidden || !selectedMetric) return;

    const canceler = new AbortController();
    const trialIds: number[] = [];
    const hpTrialMap: Record<
      string,
      Record<number, { hp: Primitive; metric: number | undefined }>
    > = {};

    setHasLoaded(false);

    readStream<V1TrialsSnapshotResponse>(
      detApi.StreamingInternal.trialsSnapshot(
        experiment.id,
        selectedMetric.name,
        selectedBatch,
        metricTypeParamMap[selectedMetric.type],
        undefined, // custom metric group
        selectedBatchMargin,
        undefined,
        { signal: canceler.signal },
      ),
      (event) => {
        if (!event?.trials || !Array.isArray(event.trials)) return;

        const hpMetricMap: Record<string, (number | undefined)[]> = {};
        const hpValueMap: Record<string, number[]> = {};
        const hpLabelMap: Record<string, string[]> = {};
        const hpLogScaleMap: Record<string, boolean> = {};

        event.trials.forEach((trial) => {
          const trialId = trial.trialId;
          trialIds.push(trialId);

          const flatHParams = flattenObject(trial.hparams);
          fullHParams.forEach((hParam) => {
            /**
             * TODO: filtering NaN, +/- Infinity for now, but handle it later with
             * dynamic min/max ranges via uPlot.Scales.
             */
            const trialMetric = Number.isFinite(trial.metric) ? trial.metric : undefined;
            hpTrialMap[hParam] = hpTrialMap[hParam] || {};
            hpTrialMap[hParam][trialId] = hpTrialMap[hParam][trialId] || {};
            hpTrialMap[hParam][trialId] = {
              hp: flatHParams[hParam],
              metric: trialMetric,
            };
          });
        });

        fullHParams.forEach((hParam) => {
          const hp = experiment.hyperparameters?.[hParam];
          if (hp.type === HyperparameterType.Log) hpLogScaleMap[hParam] = true;

          hpMetricMap[hParam] = [];
          hpValueMap[hParam] = [];
          hpLabelMap[hParam] = [];
          trialIds.forEach((trialId) => {
            const map = hpTrialMap[hParam]?.[trialId] || {};
            const hpValue = isBoolean(map.hp) ? map.hp.toString() : map.hp;

            if (isString(hpValue)) {
              // Handle categorical hp.
              let hpLabelIndex = hpLabelMap[hParam].indexOf(hpValue);
              if (hpLabelIndex === -1) {
                hpLabelIndex = hpLabelMap[hParam].length;
                hpLabelMap[hParam].push(hpValue);
              }
              hpValueMap[hParam].push(hpLabelIndex);
            } else {
              hpValueMap[hParam].push(hpValue);
            }

            hpMetricMap[hParam].push(map.metric);
          });
        });

        setChartData({
          hpLabels: hpLabelMap,
          hpLogScales: hpLogScaleMap,
          hpValues: hpValueMap,
          metricValues: hpMetricMap,
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
    experiment,
    fullHParams,
    selectedBatch,
    selectedBatchMargin,
    selectedMetric,
    ui.isPageHidden,
  ]);

  useEffect(() => setGalleryHeight(resize.height), [resize]);

  if (pageError) {
    return <Message title={pageError.message} />;
  } else if (hasLoaded && !chartData) {
    return isExperimentTerminal ? (
      <Message title="No data to plot." type={MessageType.Empty} />
    ) : (
      <div>
        <Alert
          description="Please wait until the experiment is further along."
          message="Not enough data points to plot."
        />
        <Spinner />
      </div>
    );
  }

  return (
    <div className={css.base} ref={baseRef}>
      <Section
        bodyBorder
        bodyNoPadding
        bodyScroll
        filters={filters}
        loading={!hasLoaded || !chartData}>
        <div className={css.container}>
          {chartProps ? (
            <Grid
              border={true}
              minItemWidth={resize.width > 320 ? 350 : 270}
              mode={GridMode.AutoFill}>
              {selectedHParams.map((hParam) => (
                <div key={hParam} onClick={() => handleChartClick(hParam)}>
                  <UPlotScatter
                    data={chartProps[hParam].data}
                    options={chartProps[hParam].options}
                    tooltipLabels={chartProps[hParam].tooltipLabels}
                  />
                </div>
              ))}
            </Grid>
          ) : (
            <Message title="No data to plot." type={MessageType.Empty} />
          )}
        </div>
      </Section>
      <GalleryModal
        height={galleryHeight}
        open={!!activeHParam}
        onCancel={handleGalleryClose}
        onNext={handleGalleryNext}
        onPrevious={handleGalleryPrevious}>
        {chartProps && activeHParam && (
          <UPlotScatter
            data={chartProps[activeHParam].data}
            options={{
              ...chartProps[activeHParam].options,
              cursor: { drag: undefined },
              height: galleryHeight,
            }}
            tooltipLabels={chartProps[activeHParam].tooltipLabels}
          />
        )}
      </GalleryModal>
    </div>
  );
};

export default ScatterPlots;
