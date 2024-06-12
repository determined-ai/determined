import Message from 'hew/Message';
import { useModal } from 'hew/Modal';
import Spinner from 'hew/Spinner';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import GalleryModalComponent from 'components/GalleryModalComponent';
import Grid, { GridMode } from 'components/Grid';
import { UPlotScatterProps } from 'components/UPlot/types';
import UPlotScatter from 'components/UPlot/UPlotScatter';
import { RunMetricData } from 'hooks/useMetrics';
import useResize from 'hooks/useResize';
import {
  ExperimentWithTrial,
  FlatRun,
  Hyperparameter,
  HyperparameterType,
  Primitive,
  Scale,
  TrialHyperparameters,
  TrialItem,
  XAxisDomain,
  XOR,
} from 'types';
import { flattenObject, isBoolean, isString } from 'utils/data';
import { metricToKey, metricToStr } from 'utils/metric';

import { CompareHyperparametersSettings } from './CompareHyperparameters.settings';

export const COMPARE_SCATTER_PLOTS = 'compare-scatterplots';

interface BaseProps {
  metricData: RunMetricData;
  fullHParams: string[];
  settings: CompareHyperparametersSettings;
}

type Props = XOR<
  { selectedExperiments: ExperimentWithTrial[]; trials: TrialItem[] },
  { selectedRuns: FlatRun[] }
> &
  BaseProps;

interface HpMetricData {
  hpLabels: Record<string, string[]>;
  hpLogScales: Record<string, boolean>;
  hpValues: Record<string, number[]>;
  metricValues: Record<string, (number | undefined)[]>;
  recordIds: number[];
}

const CompareScatterPlots: React.FC<Props> = ({
  fullHParams,
  trials,
  settings,
  metricData,
  selectedExperiments,
  selectedRuns,
}: Props) => {
  const baseRef = useRef<HTMLDivElement>(null);
  const [chartData, setChartData] = useState<HpMetricData>();
  const [activeHParam, setActiveHParam] = useState<string>();
  const galleryModal = useModal(GalleryModalComponent);

  const selectedScale = settings.scale;
  const selectedMetric = settings.metric;
  const selectedHParams = settings.hParams;

  const { data, isLoaded: metricsLoaded } = metricData;

  useEffect(() => {
    if (activeHParam) {
      galleryModal.open();
    }
  }, [activeHParam, galleryModal]);

  const yScaleKey = selectedScale === Scale.Log ? 'yLog' : 'y';

  const resize = useResize(baseRef);

  const chartProps = useMemo(() => {
    if (!chartData || !selectedMetric) return undefined;
    return selectedHParams.reduce((acc: Record<string, UPlotScatterProps>, hParam) => {
      const xLabel = hParam;
      const yLabel = metricToStr(selectedMetric, 60);
      const title = `${yLabel} (y) vs ${xLabel} (x)`;
      const hpLabels = chartData?.hpLabels[hParam];
      const isLogarithmic = chartData?.hpLogScales[hParam];
      const isCategorical = hpLabels?.length !== 0;
      const xScaleKey = isCategorical ? 'xCategorical' : isLogarithmic ? 'xLog' : 'x';
      const xSplits = isCategorical
        ? new Array(hpLabels?.length).fill(0).map((_x, i) => i)
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
            chartData?.recordIds || [],
          ],
        ],
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
        tooltipLabels: [xLabel, yLabel, null, null, null, 'record ID'],
      };
      return acc;
    }, {});
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

  const experimentHyperparameters = useMemo(() => {
    const hpMap: Record<string, Hyperparameter> = {};
    selectedExperiments?.forEach((exp) => {
      const hps = Object.keys(exp.experiment.hyperparameters);
      hps.forEach((hp) => (hpMap[hp] = exp.experiment.hyperparameters[hp]));
    });
    return hpMap;
  }, [selectedExperiments]);

  useEffect(() => {
    if (!selectedMetric) return;

    const recordIds: number[] = [];
    const hpTrialMap: Record<
      string,
      Record<number, { hp: Primitive; metric: number | undefined }>
    > = {};

    const hpMetricMap: Record<string, (number | undefined)[]> = {};
    const hpValueMap: Record<string, number[]> = {};
    const hpLabelMap: Record<string, string[]> = {};
    const hpLogScaleMap: Record<string, boolean> = {};

    const recordHyperparameters: [number, TrialHyperparameters][] = selectedRuns
      ? selectedRuns.flatMap((run) => (run.hyperparameters ? [[run.id, run.hyperparameters]] : []))
      : trials.map((trial) => [trial.id, trial.hyperparameters]);

    recordHyperparameters?.forEach(([recordId, recordHp]) => {
      recordIds.push(recordId);

      const flatHParams = flattenObject(recordHp);
      fullHParams.forEach((hParam: string) => {
        /**
         * TODO: filtering NaN, +/- Infinity for now, but handle it later with
         * dynamic min/max ranges via uPlot.Scales.
         */
        const key = metricToKey(selectedMetric);
        const trialMetric = data?.[recordId]?.[key]?.data?.[XAxisDomain.Batches]?.at(-1)?.[1];

        hpTrialMap[hParam] = hpTrialMap[hParam] || {};
        hpTrialMap[hParam][recordId] = hpTrialMap[hParam][recordId] || {};
        hpTrialMap[hParam][recordId] = {
          hp: flatHParams[hParam],
          metric: trialMetric,
        };
        const hp = experimentHyperparameters[key] || {};
        if (hp.type === HyperparameterType.Log) hpLogScaleMap[hParam] = true;

        hpMetricMap[hParam] = [];
        hpValueMap[hParam] = [];
        hpLabelMap[hParam] = [];
        recordIds.forEach((recordId) => {
          const map = hpTrialMap[hParam]?.[recordId] || {};
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
    });

    setChartData({
      hpLabels: hpLabelMap,
      hpLogScales: hpLogScaleMap,
      hpValues: hpValueMap,
      metricValues: hpMetricMap,
      recordIds,
    });
  }, [fullHParams, experimentHyperparameters, selectedMetric, trials, data, selectedRuns]);

  if (!metricsLoaded || !chartData) {
    return <Spinner center spinning />;
  }

  return (
    <div ref={baseRef}>
      <>
        {chartProps ? (
          <div data-testid={COMPARE_SCATTER_PLOTS}>
            <Grid
              border={true}
              minItemWidth={resize.width > 320 ? 350 : 290}
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
          </div>
        ) : (
          <Message icon="warning" title="No data to plot." />
        )}
      </>
      <galleryModal.Component
        activeHParam={activeHParam}
        chartProps={chartProps}
        selectedScale={selectedScale}
        onCancel={handleGalleryClose}
        onNext={handleGalleryNext}
        onPrevious={handleGalleryPrevious}
      />
    </div>
  );
};

export default CompareScatterPlots;
