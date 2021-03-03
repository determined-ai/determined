import { Alert, Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import ColorLegend from 'components/ColorLegend';
import Grid, { GridMode } from 'components/Grid';
import GridListRadioGroup, { GridListView } from 'components/GridListRadioGroup';
import Message, { MessageType } from 'components/Message';
import MetricBadgeTag from 'components/MetricBadgeTag';
import MetricSelectFilter from 'components/MetricSelectFilter';
import MultiSelect from 'components/MultiSelect';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ScatterPlot from 'components/ScatterPlot';
import Section from 'components/Section';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import useResize from 'hooks/useResize';
import useStorage from 'hooks/useStorage';
import { V1TrialsSnapshotResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import {
  ExperimentBase, ExperimentHyperParamType, MetricName, MetricType, metricTypeParamMap, Range,
} from 'types';
import { getColorScale } from 'utils/chart';
import { isNumber, isObject } from 'utils/data';
import { ConditionalWrapper } from 'utils/react';
import { metricNameToStr } from 'utils/string';
import { terminalRunStates } from 'utils/types';

import css from './HpHeatMaps.module.scss';

const { Option } = Select;

interface Props {
  batches: number[];
  experiment: ExperimentBase;
  hParams: string[];
  isLoading?: boolean;
  metrics: MetricName[];
  onBatchChange?: (batch: number) => void;
  onMetricChange?: (metric: MetricName) => void;
  selectedBatch: number;
  selectedMetric: MetricName;
}

interface HpData {
  hpLogScales: Record<string, boolean>;
  hpMetrics: Record<string, number[]>;
  hpValues: Record<string, number[]>;
  metricRange: Range<number>;
  trialIds: number[];
}

const MAX_HPARAM_COUNT = 5;
const STORAGE_PATH = 'hp-vs-hp';
const STORAGE_HPARAMS_KEY = 'hparams';
const STORAGE_VIEW_KEY = 'grid-list';

const generateHpKey = (hParam1: string, hParam2: string): string => {
  return `${hParam1}:${hParam2}`;
};

const HpHeatMaps: React.FC<Props> = ({
  batches,
  experiment,
  hParams,
  isLoading = false,
  metrics,
  onBatchChange,
  onMetricChange,
  selectedBatch,
  selectedMetric,
}: Props) => {
  const baseRef = useRef<HTMLDivElement>(null);
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ chartData, setChartData ] = useState<HpData>();
  const [ pageError, setPageError ] = useState<Error>();
  const storage = useStorage(STORAGE_PATH);
  const defaultHParams = storage.get<string[]>(STORAGE_HPARAMS_KEY);
  const limitedHParams = hParams.slice(0, MAX_HPARAM_COUNT);
  const [
    selectedHParams,
    setSelectedHParams,
  ] = useState<string[]>(defaultHParams || limitedHParams);
  const defaultView = storage.get<GridListView>(STORAGE_VIEW_KEY) || GridListView.Grid;
  const [ selectedView, setSelectedView ] = useState(defaultView);
  const resize = useResize(baseRef);
  const classes = [ css.base ];

  const isExperimentTerminal = terminalRunStates.has(experiment.state);
  const isListView = selectedView === GridListView.List;

  if (isListView) classes.push(css.list);

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

  const resetData = useCallback(() => {
    setChartData(undefined);
    setHasLoaded(false);
  }, []);

  const handleBatchChange = useCallback((batch: SelectValue) => {
    if (!onBatchChange) return;
    resetData();
    onBatchChange(batch as number);
  }, [ onBatchChange, resetData ]);

  const handleHParamChange = useCallback((hps: SelectValue) => {
    if (Array.isArray(hps) && hps.length !== 0) {
      storage.set(STORAGE_HPARAMS_KEY, hps);
      setSelectedHParams(hps as string[]);
    } else {
      storage.remove(STORAGE_HPARAMS_KEY);
      setSelectedHParams(limitedHParams);
    }
  }, [ limitedHParams, storage ]);

  const handleMetricChange = useCallback((metric: MetricName) => {
    if (!onMetricChange) return;
    resetData();
    onMetricChange(metric);
  }, [ onMetricChange, resetData ]);

  const handleViewChange = useCallback((value: GridListView) => setSelectedView(value), []);

  useEffect(() => {
    const canceler = new AbortController();

    const trialIds: number[] = [];
    const hpMetricMap: Record<number, Record<string, number>> = {};
    const hpValueMap: Record<number, Record<string, number>> = {};

    consumeStream<V1TrialsSnapshotResponse>(
      detApi.StreamingInternal.determinedTrialsSnapshot(
        experiment.id,
        selectedBatch,
        selectedMetric.name,
        metricTypeParamMap[selectedMetric.type],
        undefined,
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
    ).catch(e => setPageError(e));

    return () => canceler.abort();
  }, [ experiment, hParams, selectedBatch, selectedMetric ]);

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
  if (hasLoaded && !isLoading && chartData) {
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
            <ConditionalWrapper
              condition={isListView}
              falseWrapper={children => <div className={css.grid}>{children}</div>}
              wrapper={children => (
                <Grid
                  border={true}
                  minItemWidth={resize.width > 320 ? 35 : 27}
                  mode={GridMode.AutoFill}>{children}</Grid>
              )}>
              <>{selectedHParams.map(hParam1 => (
                <ConditionalWrapper
                  condition={!isListView}
                  key={hParam1}
                  wrapper={children => (
                    <div className={css.row} key={hParam1}>{children}</div>
                  )}>
                  <>{selectedHParams.map(hParam2 => {
                    const key = generateHpKey(hParam1, hParam2);
                    return (
                      <ConditionalWrapper
                        condition={!isListView}
                        key={hParam2}
                        wrapper={children => (
                          <div className={css.item} key={hParam2}>{children}</div>
                        )}>
                        <ScatterPlot
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
                        />
                      </ConditionalWrapper>
                    );
                  })}</>
                </ConditionalWrapper>
              ))}</>
            </ConditionalWrapper>
          </div>
        </>
      );
    }
  }

  return (
    <div className={classes.join(' ')} ref={baseRef}>
      <Section
        options={<ResponsiveFilters>
          <SelectFilter
            enableSearchFilter={false}
            label="Batches Processed"
            showSearch={false}
            value={selectedBatch}
            onChange={handleBatchChange}>
            {batches.map(batch => <Option key={batch} value={batch}>{batch}</Option>)}
          </SelectFilter>
          <MetricSelectFilter
            defaultMetricNames={metrics}
            label="Metric"
            metricNames={metrics}
            multiple={false}
            value={selectedMetric}
            width={'100%'}
            onChange={handleMetricChange} />
          <MultiSelect
            label="HP"
            value={selectedHParams}
            onChange={handleHParamChange}>
            {hParams.map(hpKey => <Option key={hpKey} value={hpKey}>{hpKey}</Option>)}
          </MultiSelect>
          <GridListRadioGroup value={selectedView} onChange={handleViewChange} />
        </ResponsiveFilters>}
        title="HP Heat Maps">
        <div className={css.container}>{content}</div>
      </Section>
    </div>
  );
};

export default HpHeatMaps;
