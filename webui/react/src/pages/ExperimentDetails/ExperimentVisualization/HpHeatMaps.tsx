import { Alert } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import ColorLegend from 'components/ColorLegend';
import GalleryModal from 'components/GalleryModal';
import Grid, { GridMode } from 'components/Grid';
import Message, { MessageType } from 'components/Message';
import MetricBadgeTag from 'components/MetricBadgeTag';
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
import { Primitive, Range } from 'types';
import {
  ExperimentBase,
  HyperparameterType,
  Metric,
  MetricType,
  metricTypeParamMap,
  Scale,
} from 'types';
import { getColorScale } from 'utils/chart';
import { rgba2str, str2rgba } from 'utils/color';
import { clone, flattenObject, isBoolean, isObject, isString } from 'utils/data';
import { metricToStr } from 'utils/metric';

import { ViewType } from './ExperimentVisualizationFilters';
import css from './HpHeatMaps.module.scss';

interface Props {
  experiment: ExperimentBase;
  filters?: React.ReactNode;
  fullHParams: string[];
  selectedBatch: number;
  selectedBatchMargin: number;
  selectedHParams: string[];
  selectedMetric?: Metric;
  selectedScale: Scale;
  selectedView?: ViewType;
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

const HpHeatMaps: React.FC<Props> = ({
  experiment,
  filters,
  fullHParams,
  selectedBatch,
  selectedBatchMargin,
  selectedHParams,
  selectedMetric,
  selectedView = ViewType.Grid,
  selectedScale,
}: Props) => {
  const { ui } = useUI();
  const baseRef = useRef<HTMLDivElement>(null);
  const resize = useResize(baseRef);
  const [hasLoaded, setHasLoaded] = useState(false);
  const [chartData, setChartData] = useState<HpData>();
  const [pageError, setPageError] = useState<Error>();
  const [activeHParam, setActiveHParam] = useState<string>();
  const [galleryHeight, setGalleryHeight] = useState<number>(450);

  const isExperimentTerminal = terminalRunStates.has(experiment.state);
  const isListView = selectedView === ViewType.List;

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

  const colorScale = useMemo(() => {
    return getColorScale(ui.theme, chartData?.metricRange, smallerIsBetter);
  }, [chartData, smallerIsBetter, ui.theme]);

  const chartProps = useMemo(() => {
    if (!chartData || !selectedMetric) return undefined;

    const props: Record<string, UPlotScatterProps> = {};
    const rgbaStroke0 = str2rgba(colorScale[0].color);
    const rgbaStroke1 = str2rgba(colorScale[1].color);
    const rgbaFill0 = clone(rgbaStroke0);
    const rgbaFill1 = clone(rgbaStroke1);
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
          ? new Array(xHpLabels.length).fill(0).map((x, i) => i)
          : undefined;
        const ySplits = isYCategorical
          ? new Array(yHpLabels.length).fill(0).map((x, i) => i)
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
          ] as FacetedData,
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

  useEffect(() => {
    if (ui.isPageHidden || !selectedMetric) return;

    const canceler = new AbortController();
    const trialIds: number[] = [];
    const hpMetricMap: Record<number, Record<string, number | undefined>> = {};
    const hpValueMap: Record<number, Record<string, Primitive>> = {};
    const hpLabelMap: Record<string, string[]> = {};
    const hpLabelValueMap: Record<string, number[]> = {};

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

        const hpLogScaleMap: Record<string, boolean> = {};
        const hpMetrics: Record<string, (number | undefined)[]> = {};
        const hpValues: HpValue = {};
        const metricRange: Range<number> = [Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY];

        event.trials.forEach((trial) => {
          if (!isObject(trial.hparams)) return;

          const trialId = trial.trialId;
          const flatHParams = flattenObject(trial.hparams);
          const trialHParams = Object.keys(flatHParams)
            .filter((hParam) => fullHParams.includes(hParam))
            .sort((a, b) => a.localeCompare(b, 'en'));

          /**
           * TODO: filtering NaN, +/- Infinity for now, but handle it later with
           * dynamic min/max ranges via uPlot.Scales.
           */
          const trialMetric = Number.isFinite(trial.metric) ? trial.metric : undefined;

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

          if (trialMetric !== undefined && trialMetric < metricRange[0])
            metricRange[0] = trialMetric;
          if (trialMetric !== undefined && trialMetric > metricRange[1])
            metricRange[1] = trialMetric;
        });

        fullHParams.forEach((hParam1) => {
          const hp = experiment.hyperparameters?.[hParam1];
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
  } else if ((hasLoaded && !chartData) || !selectedMetric) {
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
      <Section bodyBorder bodyNoPadding bodyScroll filters={filters} loading={!hasLoaded}>
        <div className={css.container}>
          {chartProps ? (
            <>
              <div className={css.legend}>
                <ColorLegend
                  colorScale={colorScale}
                  title={<MetricBadgeTag metric={selectedMetric} />}
                />
              </div>
              <div className={css.charts}>
                <Grid
                  border={true}
                  minItemWidth={resize.width > 320 ? 350 : 270}
                  mode={!isListView ? selectedHParams.length : GridMode.AutoFill}>
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
            colorScaleDistribution={selectedScale}
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

export default HpHeatMaps;
