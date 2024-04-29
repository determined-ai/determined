import Message from 'hew/Message';
import { useModal } from 'hew/Modal';
import { isObject } from 'lodash';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import ColorLegend from 'components/ColorLegend';
import GalleryModalComponent from 'components/GalleryModalComponent';
import Grid, { GridMode } from 'components/Grid';
import MetricBadgeTag from 'components/MetricBadgeTag';
import useUI from 'components/ThemeProvider';
import { UPlotScatterProps } from 'components/UPlot/types';
import UPlotScatter from 'components/UPlot/UPlotScatter';
import useResize from 'hooks/useResize';
import { TrialMetricData } from 'pages/TrialDetails/useTrialMetrics';
import {
  ExperimentWithTrial,
  Hyperparameter,
  HyperparameterType,
  MetricType,
  Primitive,
  Range,
  TrialItem,
  XAxisDomain,
} from 'types';
import { getColorScale } from 'utils/chart';
import { rgba2str, str2rgba } from 'utils/color';
import { flattenObject, isBoolean, isString } from 'utils/data';
import { metricToKey, metricToStr } from 'utils/metric';

import { CompareHyperparametersSettings } from './CompareHyperparameters.settings';

interface Props {
  selectedExperiments: ExperimentWithTrial[];
  trials: TrialItem[];
  metricData: TrialMetricData;
  fullHParams: string[];
  settings: CompareHyperparametersSettings;
}

type HpValue = Record<string, (number | string)[]>;

interface HpData {
  hpLabelValues: Record<string, number[]>;
  hpLabels: Record<string, string[]>;
  hpLogScales: Record<string, boolean>;
  hpMetrics: Record<string, (number | undefined)[]>;
  hpValues: HpValue;
  metricRange: Range<number>;
  trialIds: number[];
}

const generateHpKey = (hParam1: string, hParam2: string): string => {
  return `${hParam1}:${hParam2}`;
};

const parseHpKey = (key: string): [hParam1: string, hParam2: string] => {
  const parts = key.split(':');
  return [parts[0], parts[1]];
};

const CompareHeatMaps: React.FC<Props> = ({
  selectedExperiments,
  trials,
  metricData,
  fullHParams,
  settings,
}: Props) => {
  const { ui } = useUI();
  const baseRef = useRef<HTMLDivElement>(null);
  const resize = useResize(baseRef);
  const [chartData, setChartData] = useState<HpData>();
  const [activeHParam, setActiveHParam] = useState<string>();
  const galleryModal = useModal(GalleryModalComponent);

  const selectedScale = settings.scale;
  const selectedMetric = settings.metric;
  const selectedHParams = settings.hParams;

  const { data } = metricData;

  useEffect(() => {
    if (activeHParam) {
      galleryModal.open();
    }
  }, [activeHParam, galleryModal]);

  const smallerIsBetter = useMemo(() => {
    if (selectedMetric && selectedMetric.group === MetricType.Validation) {
      const selectedExperimentsWithMetric = selectedExperiments.filter((exp) => {
        return selectedMetric.name === exp?.experiment?.config?.searcher?.metric;
      });

      return selectedExperimentsWithMetric.some((exp) => {
        return exp?.experiment?.config?.searcher?.smallerIsBetter;
      });
    }
    return undefined;
  }, [selectedMetric, selectedExperiments]);

  const colorScale = useMemo(() => {
    return getColorScale(ui.theme, chartData?.metricRange, smallerIsBetter);
  }, [chartData, smallerIsBetter, ui.theme]);

  const chartProps = useMemo(() => {
    if (!chartData || !selectedMetric) return undefined;

    const props: Record<string, UPlotScatterProps> = {};
    const rgbaStroke0 = str2rgba(colorScale[0].color);
    const rgbaStroke1 = str2rgba(colorScale[1].color);
    const rgbaFill0 = structuredClone(rgbaStroke0);
    const rgbaFill1 = structuredClone(rgbaStroke1);
    rgbaFill0.a = 0.3;
    rgbaFill1.a = 0.3;
    const fill = [rgba2str(rgbaFill0), rgba2str(rgbaFill1)].join(' ');
    const stroke = [rgba2str(rgbaStroke0), rgba2str(rgbaStroke1)].join(' ');

    selectedHParams.forEach((hParam1) => {
      selectedHParams.forEach((hParam2) => {
        const key = generateHpKey(hParam1, hParam2);
        const xLabel = hParam2;
        const yLabel = hParam1;
        const title = `${yLabel} (y) vs ${xLabel} (x)`;
        const xHpLabels = chartData?.hpLabels[hParam2];
        const yHpLabels = chartData?.hpLabels[hParam1];
        const isXLogarithmic = chartData?.hpLogScales[hParam2];
        const isYLogarithmic = chartData?.hpLogScales[hParam1];
        const isXCategorical = xHpLabels?.length !== 0;
        const isYCategorical = yHpLabels?.length !== 0;
        const xScaleKey = isXCategorical ? 'xCategorical' : isXLogarithmic ? 'xLog' : 'x';
        const yScaleKey = isYCategorical ? 'yCategorical' : isYLogarithmic ? 'yLog' : 'y';
        const xSplits = isXCategorical
          ? new Array(xHpLabels.length).fill(0).map((_x, i) => i)
          : undefined;
        const ySplits = isYCategorical
          ? new Array(yHpLabels.length).fill(0).map((_x, i) => i)
          : undefined;
        const xValues = isXCategorical ? xHpLabels : undefined;
        const yValues = isYCategorical ? yHpLabels : undefined;

        props[key] = {
          data: [
            null,
            [
              chartData?.hpValues[hParam2] || [],
              chartData?.hpValues[hParam1] || [],
              null,
              chartData?.hpMetrics[key] || [],
              chartData?.hpMetrics[key] || [],
              chartData?.trialIds || [],
            ],
          ],
          options: {
            axes: [
              { scale: xScaleKey, splits: xSplits, values: xValues },
              { scale: yScaleKey, splits: ySplits, values: yValues },
            ],
            cursor: { drag: { setScale: false, x: false, y: false } },
            series: [{}, { fill, stroke }],
            title,
          },
          tooltipLabels: [xLabel, yLabel, null, metricToStr(selectedMetric), null, 'trial ID'],
        };
      });
    });

    return props;
  }, [chartData, colorScale, selectedHParams, selectedMetric]);

  const handleChartClick = useCallback((hParam1: string, hParam2: string) => {
    setActiveHParam(generateHpKey(hParam1, hParam2));
  }, []);

  const handleGalleryClose = useCallback(() => setActiveHParam(undefined), []);

  const handleGalleryNext = useCallback(() => {
    setActiveHParam((prev) => {
      if (!prev) return prev;
      const [hParam1, hParam2] = parseHpKey(prev);
      const index0 = selectedHParams.indexOf(hParam1);
      const index1 = selectedHParams.indexOf(hParam2);
      if (index0 === -1 || index1 === -1) return prev;
      if (index0 === selectedHParams.length - 1 && index1 === selectedHParams.length - 1) {
        return generateHpKey(selectedHParams[0], selectedHParams[0]);
      } else if (index1 === selectedHParams.length - 1) {
        return generateHpKey(selectedHParams[index0 + 1], selectedHParams[0]);
      } else {
        return generateHpKey(selectedHParams[index0], selectedHParams[index1 + 1]);
      }
    });
  }, [selectedHParams]);

  const handleGalleryPrevious = useCallback(() => {
    setActiveHParam((prev) => {
      if (!prev) return prev;
      const [hParam1, hParam2] = parseHpKey(prev);
      const index0 = selectedHParams.indexOf(hParam1);
      const index1 = selectedHParams.indexOf(hParam2);
      if (index0 === -1 || index1 === -1) return prev;
      if (index0 === 0 && index1 === 0) {
        return generateHpKey(selectedHParams.last(), selectedHParams.last());
      } else if (index1 === 0) {
        return generateHpKey(selectedHParams[index0 - 1], selectedHParams.last());
      } else {
        return generateHpKey(selectedHParams[index0], selectedHParams[index1 - 1]);
      }
    });
  }, [selectedHParams]);

  const experimentHyperparameters = useMemo(() => {
    const hpMap: Record<string, Hyperparameter> = {};
    selectedExperiments.forEach((exp) => {
      const hps = Object.keys(exp.experiment.hyperparameters);
      hps.forEach((hp) => (hpMap[hp] = exp.experiment.hyperparameters[hp]));
    });
    return hpMap;
  }, [selectedExperiments]);

  useEffect(() => {
    if (ui.isPageHidden || !selectedMetric) return;

    const trialIds: number[] = [];
    const hpMetricMap: Record<number, Record<string, number | undefined>> = {};
    const hpValueMap: Record<number, Record<string, Primitive>> = {};
    const hpLabelMap: Record<string, string[]> = {};
    const hpLabelValueMap: Record<string, number[]> = {};

    const hpLogScaleMap: Record<string, boolean> = {};
    const hpMetrics: Record<string, (number | undefined)[]> = {};
    const hpValues: HpValue = {};
    const metricRange: Range<number> = [Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY];

    trials.forEach((trial) => {
      if (!isObject(trial.hyperparameters)) return;

      const trialId = trial.id;
      const flatHParams = flattenObject(trial.hyperparameters);
      const trialHParams = Object.keys(flatHParams)
        .filter((hParam) => fullHParams.includes(hParam))
        .sort((a, b) => a.localeCompare(b, 'en'));

      /**
       * TODO: filtering NaN, +/- Infinity for now, but handle it later with
       * dynamic min/max ranges via uPlot.Scales.
       */
      const key = metricToKey(selectedMetric);
      const trialMetric = data?.[trial.id]?.[key]?.data?.[XAxisDomain.Batches]?.at(-1)?.[1];

      trialIds.push(trialId);
      hpMetricMap[trialId] = hpMetricMap[trialId] || {};
      hpValueMap[trialId] = hpValueMap[trialId] || {};
      trialHParams.forEach((hParam1) => {
        hpValueMap[trialId][hParam1] = flatHParams[hParam1];
        trialHParams.forEach((hParam2) => {
          const key = generateHpKey(hParam1, hParam2);
          hpMetricMap[trialId][key] = trialMetric;
        });
      });

      if (trialMetric !== undefined && trialMetric < metricRange[0]) metricRange[0] = trialMetric;
      if (trialMetric !== undefined && trialMetric > metricRange[1]) metricRange[1] = trialMetric;
    });

    fullHParams.forEach((hParam1) => {
      const key = metricToKey(selectedMetric);
      const hp = experimentHyperparameters[key] || {};
      if (hp.type === HyperparameterType.Log) hpLogScaleMap[hParam1] = true;

      hpLabelMap[hParam1] = [];
      hpLabelValueMap[hParam1] = [];
      hpValues[hParam1] = [];

      trialIds.forEach((trialId) => {
        const hpRawValue = hpValueMap[trialId][hParam1];
        const hpValue = isBoolean(hpRawValue) ? hpRawValue.toString() : hpRawValue;

        hpValues[hParam1].push(hpValue);

        if (isString(hpValue)) {
          // Handle categorical hp.
          let hpLabelIndex = hpLabelMap[hParam1].indexOf(hpValue);
          if (hpLabelIndex === -1) {
            hpLabelIndex = hpLabelMap[hParam1].length;
            hpLabelMap[hParam1].push(hpValue);
          }
          hpLabelValueMap[hParam1].push(hpLabelIndex);
        } else {
          hpLabelValueMap[hParam1].push(hpValue);
        }
      });

      fullHParams.forEach((hParam2) => {
        const key = generateHpKey(hParam1, hParam2);
        hpMetrics[key] = trialIds.map((trialId) => hpMetricMap[trialId][key]);
      });
    });

    setChartData({
      hpLabels: hpLabelMap,
      hpLabelValues: hpLabelValueMap,
      hpLogScales: hpLogScaleMap,
      hpMetrics,
      hpValues,
      metricRange,
      trialIds,
    });
  }, [fullHParams, selectedMetric, ui.isPageHidden, trials, data, experimentHyperparameters]);

  return (
    <div ref={baseRef}>
      <div>
        {chartProps && selectedMetric ? (
          <>
            <div>
              <ColorLegend
                colorScale={colorScale}
                title={<MetricBadgeTag metric={selectedMetric} />}
              />
            </div>
            <div>
              <Grid
                border={true}
                minItemWidth={resize.width > 320 ? 350 : 270}
                mode={GridMode.AutoFill}>
                {selectedHParams.map((hParam1) =>
                  selectedHParams.map((hParam2) => {
                    const key = generateHpKey(hParam1, hParam2);
                    return (
                      <div key={key} onClick={() => handleChartClick(hParam1, hParam2)}>
                        <UPlotScatter
                          colorScaleDistribution={selectedScale}
                          data={chartProps[key].data}
                          options={chartProps[key].options}
                          tooltipLabels={chartProps[key].tooltipLabels}
                        />
                      </div>
                    );
                  }),
                )}
              </Grid>
            </div>
          </>
        ) : (
          <Message icon="warning" title="No data to plot." />
        )}
      </div>
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

export default CompareHeatMaps;
