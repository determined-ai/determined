import { Alert } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import ColorLegend from 'components/ColorLegend';
import GalleryModal from 'components/GalleryModal';
import Grid, { GridMode } from 'components/Grid';
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
import { isObject } from 'utils/data';
import { metricNameToStr } from 'utils/string';
import { terminalRunStates } from 'utils/types';

import css from './HpHeatMaps.module.scss';

interface Props {
  experiment: ExperimentBase;
  filters?: React.ReactNode;
  fullHParams: string[];
  selectedBatch: number;
  selectedBatchMargin: number;
  selectedHParams: string[];
  selectedMetric: MetricName;
  selectedView: ViewType;
}

interface HpData {
  hpLogScales: Record<string, boolean>;
  hpMetrics: Record<string, number[]>;
  hpValues: Record<string, number[]>;
  metricRange: Range<number>;
  trialIds: number[];
}

enum ViewType {
  Grid = 'grid',
  List = 'list',
}

const generateHpKey = (hParam1: string, hParam2: string): string => {
  return `${hParam1}:${hParam2}`;
};

const HpHeatMaps: React.FC<Props> = ({
  experiment,
  filters,
  fullHParams,
  selectedBatch,
  selectedBatchMargin,
  selectedHParams,
  selectedMetric,
  selectedView,
}: Props) => {
  const baseRef = useRef<HTMLDivElement>(null);
  const resize = useResize(baseRef);
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ chartData, setChartData ] = useState<HpData>();
  const [ pageError, setPageError ] = useState<Error>();
  const [ activeHParam, setActiveHParam ] = useState<[ string, string ]>();
  const [ galleryHeight, setGalleryHeight ] = useState<number>(450);

  const isExperimentTerminal = terminalRunStates.has(experiment.state);
  const isListView = selectedView === ViewType.List;

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

  const handleChartClick = useCallback((hParam1: string, hParam2: string) => {
    setActiveHParam([ hParam1, hParam2 ]);
  }, []);

  const handleGalleryClose = useCallback(() => setActiveHParam(undefined), []);

  const handleGalleryNext = useCallback(() => {
    setActiveHParam(prev => {
      if (!prev) return prev;
      const index0 = selectedHParams.indexOf(prev[0]);
      const index1 = selectedHParams.indexOf(prev[1]);
      if (index0 === -1 || index1 === -1) return prev;
      if (index0 === selectedHParams.length - 1 && index1 === selectedHParams.length - 1) {
        return [ selectedHParams[0], selectedHParams[0] ];
      } else if (index1 === selectedHParams.length - 1) {
        return [ selectedHParams[index0 + 1], selectedHParams[0] ];
      } else {
        return [ selectedHParams[index0], selectedHParams[index1 + 1] ];
      }
    });
  }, [ selectedHParams ]);

  const handleGalleryPrevious = useCallback(() => {
    setActiveHParam(prev => {
      if (!prev) return prev;
      const index0 = selectedHParams.indexOf(prev[0]);
      const index1 = selectedHParams.indexOf(prev[1]);
      if (index0 === -1 || index1 === -1) return prev;
      if (index0 === 0 && index1 === 0) {
        return [ selectedHParams.last(), selectedHParams.last() ];
      } else if (index1 === 0) {
        return [ selectedHParams[index0 - 1], selectedHParams.last() ];
      } else {
        return [ selectedHParams[index0], selectedHParams[index1 - 1] ];
      }
    });
  }, [ selectedHParams ]);

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
            .filter(hParam => fullHParams.includes(hParam))
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

        fullHParams.forEach(hParam1 => {
          const hp = (experiment.config.hyperparameters || {})[hParam1];
          if (hp.type === ExperimentHyperParamType.Log) hpLogScaleMap[hParam1] = true;

          hpValues[hParam1] = trialIds.map(trialId => hpValueMap[trialId][hParam1]);
          fullHParams.forEach(hParam2 => {
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
      <Section
        bodyBorder
        bodyNoPadding
        bodyScroll
        filters={filters}
        loading={!hasLoaded}>
        <div className={css.container}>
          {chartData?.trialIds.length === 0 ? (
            <Message title="No data to plot." type={MessageType.Empty} />
          ) : (
            <>
              <div className={css.legend}>
                <ColorLegend
                  colorScale={colorScale}
                  title={<MetricBadgeTag metric={selectedMetric} />} />
              </div>
              <div className={css.charts}>
                <Grid
                  border={true}
                  minItemWidth={resize.width > 320 ? 350 : 270}
                  mode={!isListView ? selectedHParams.length : GridMode.AutoFill}>
                  {selectedHParams.map(hParam1 => selectedHParams.map(hParam2 => {
                    const key = generateHpKey(hParam1, hParam2);
                    return (
                      <div key={key} onClick={() => handleChartClick(hParam1, hParam2)}>
                        <ScatterPlot
                          colorScale={colorScale}
                          disableZoom
                          height={350}
                          valueLabel={metricNameToStr(selectedMetric)}
                          values={chartData?.hpMetrics[key]}
                          width={350}
                          x={chartData?.hpValues[hParam2] || []}
                          xLabel={hParam2}
                          xLogScale={chartData?.hpLogScales[hParam2]}
                          y={chartData?.hpValues[hParam1] || []}
                          yLabel={hParam1}
                          yLogScale={chartData?.hpLogScales[hParam1]}
                        />
                      </div>
                    );
                  }))}
                </Grid>
              </div>
            </>
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
          colorScale={colorScale}
          height={galleryHeight}
          valueLabel={metricNameToStr(selectedMetric)}
          values={chartData?.hpMetrics[generateHpKey(activeHParam[0], activeHParam[1])]}
          width={350}
          x={chartData?.hpValues[activeHParam[1]] || []}
          xLabel={activeHParam[1]}
          xLogScale={chartData?.hpLogScales[activeHParam[1]]}
          y={chartData?.hpValues[activeHParam[0]] || []}
          yLabel={activeHParam[0]}
          yLogScale={chartData?.hpLogScales[activeHParam[0]]}
        />}
      </GalleryModal>
    </div>
  );
};

export default HpHeatMaps;
