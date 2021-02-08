import { Alert, Select } from 'antd';
import { SelectValue } from 'antd/lib/select';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Message, { MessageType } from 'components/Message';
import MetricSelectFilter from 'components/MetricSelectFilter';
import MultiSelect from 'components/MultiSelect';
import ParallelCoordinates, {
  Dimension, DimensionType, dimensionTypeMap,
} from 'components/ParallelCoordinates';
import ResponsiveFilters from 'components/ResponsiveFilters';
import Section from 'components/Section';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import useStorage from 'hooks/useStorage';
import { handlePath, paths } from 'routes/utils';
import { V1TrialsSnapshotResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import {
  ExperimentBase, ExperimentHyperParamType, MetricName, MetricType, metricTypeParamMap,
  Point,
  Primitive, Range, RunState,
} from 'types';
import { defaultNumericRange, getNumericRange, normalizeRange, updateRange } from 'utils/chart';
import { isObject } from 'utils/data';
import { terminalRunStates } from 'utils/types';

import css from './HpParallelCoordinates.module.scss';

const { Option } = Select;

interface Props {
  batches: number[];
  experiment: ExperimentBase;
  metrics: MetricName[];
  onBatchChange?: (batch: number) => void;
  onMetricChange?: (metric: MetricName) => void;
  selectedBatch: number;
  selectedMetric: MetricName;
}

interface HpTrialData {
  colors: number[];
  data: Record<string, Primitive[]>;
  metricRange?: Range<number>;
  metricValues: number[];
  trialIds: number[];
}

const STORAGE_PATH = 'experiment-visualization';
const STORAGE_HP_KEY = 'hps';
const MAX_HP_COUNT = 20;

const HpParallelCoordinates: React.FC<Props> = ({
  batches,
  experiment,
  metrics,
  onBatchChange,
  onMetricChange,
  selectedBatch,
  selectedMetric,
}: Props) => {
  const tooltipRef = useRef<HTMLDivElement>(null);
  const trialIdRef = useRef<HTMLDivElement>(null);
  const metricValueRef = useRef<HTMLDivElement>(null);
  const storage = useStorage(`${STORAGE_PATH}/${experiment.id}/parcoords`);
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ chartData, setChartData ] = useState<HpTrialData>();
  const [ pageError, setPageError ] = useState<Error>();
  const fullHpList = Object.keys(experiment.config.hyperparameters) || [];
  const limitedHpList = fullHpList.slice(0, MAX_HP_COUNT);
  const defaultHpList = storage.get<string[]>(STORAGE_HP_KEY);
  const [ hpList, setHpList ] = useState<string[]>(defaultHpList || limitedHpList);
  const [ hoverTrialId, setHoverTrialId ] = useState<number>();

  const isExperimentTerminal = terminalRunStates.has(experiment.state as RunState);

  const smallerIsBetter = useMemo(() => {
    if (selectedMetric.type === MetricType.Validation &&
        selectedMetric.name === experiment.config.searcher.metric) {
      return experiment.config.searcher.smallerIsBetter;
    }
    return undefined;
  }, [ experiment.config.searcher, selectedMetric ]);

  const dimensions = useMemo(() => {
    const newDimensions = hpList
      .filter(key => {
        const hp = experiment.config.hyperparameters[key];
        return hp.type !== ExperimentHyperParamType.Constant;
      })
      .map(key => {
        const hp = experiment.config.hyperparameters[key];
        const dimension: Dimension = {
          categories: hp.vals,
          label: key,
          type: dimensionTypeMap[hp.type],
        };

        if (hp.minval != null && hp.maxval != null) {
          const isLogarithmic = hp.type === ExperimentHyperParamType.Log;
          dimension.range = isLogarithmic ?
            [ 10 ** hp.minval, 10 ** hp.maxval ] : [ hp.minval, hp.maxval ];
        }

        return dimension;
      });

    // Add metric as column to parcoords dimension list
    if (chartData?.metricRange) {
      newDimensions.push({
        label: selectedMetric.name,
        range: chartData.metricRange,
        type: DimensionType.Scalar,
      });
    }

    return newDimensions;
  }, [ chartData, experiment.config.hyperparameters, hpList, selectedMetric.name ]);

  const resetData = useCallback(() => {
    setChartData(undefined);
    setHasLoaded(false);
  }, []);

  const handleBatchChange = useCallback((batch: SelectValue) => {
    if (!onBatchChange) return;
    resetData();
    onBatchChange(batch as number);
  }, [ onBatchChange, resetData ]);

  const handleMetricChange = useCallback((metric: MetricName) => {
    if (!onMetricChange) return;
    resetData();
    onMetricChange(metric);
  }, [ onMetricChange, resetData ]);

  const handleHpChange = useCallback((hps: SelectValue) => {
    if (Array.isArray(hps) && hps.length === 0) {
      storage.remove(STORAGE_HP_KEY);
      setHpList(limitedHpList);
    } else {
      storage.set(STORAGE_HP_KEY, hps);
      setHpList(hps as string[]);
    }
  }, [ limitedHpList, storage ]);

  const handleChartClick = useCallback((event: React.MouseEvent) => {
    if (!hoverTrialId) return;
    handlePath(event, { path: paths.trialDetails(hoverTrialId, experiment.id) });
  }, [ experiment.id, hoverTrialId ]);

  const handleChartHover = useCallback((lineIndex: number, point: Point) => {
    if (!tooltipRef.current || !trialIdRef.current || !metricValueRef.current) return;

    const trialId = chartData?.trialIds[lineIndex];
    const metricValue = chartData?.metricValues[lineIndex];
    if (!trialId || !metricValue) return;

    setHoverTrialId(trialId);
    trialIdRef.current.innerText = trialId.toString();
    metricValueRef.current.innerText = metricValue.toString();
    tooltipRef.current.style.display = 'block';
    tooltipRef.current.style.left = `${point.x}px`;
    tooltipRef.current.style.top = `${point.y}px`;
  }, [ chartData ]);

  const handleChartUnhover = useCallback(() => {
    if (!tooltipRef.current) return;

    setHoverTrialId(undefined);
    tooltipRef.current.style.display = 'none';
  }, []);

  useEffect(() => {
    const canceler = new AbortController();

    consumeStream<V1TrialsSnapshotResponse>(
      detApi.StreamingInternal.determinedTrialsSnapshot(
        experiment.id,
        selectedBatch,
        selectedMetric.name,
        metricTypeParamMap[selectedMetric.type],
        undefined,
        { signal: canceler.signal },
      ),
      event => {
        if (!event || !event.trials || !isObject(event.trials) ||
          Object.keys(event.trials).length === 0) return;

        const trialIds: number[] = [];
        const trialMetrics: number[] = [];
        const trialHpMap: Record<string, Record<number, Primitive>> = {};
        const trialHpRanges: Record<string, Range> = {};
        const data: Record<string, Primitive[]> = {};
        let trialMetricRange: Range<number> = defaultNumericRange();

        event.trials.forEach(trial => {
          trialIds.push(trial.trialId);
          trialMetrics.push(trial.metric);
          trialMetricRange = updateRange<number>(trialMetricRange, trial.metric);

          Object.keys(trial.hparams || {}).forEach(hpKey => {
            const hpValue = trial.hparams[hpKey];
            trialHpMap[hpKey] = trialHpMap[hpKey] || {};
            trialHpMap[hpKey][trial.trialId] = hpValue;
            trialHpRanges[hpKey] = updateRange(trialHpRanges[hpKey], hpValue);
          });
        });

        Object.keys(trialHpMap).forEach(hpKey => {
          data[hpKey] = trialIds.map(trialId => trialHpMap[hpKey][trialId]);
        });

        // Add metric of interest
        data[selectedMetric.name] = trialMetrics;

        // Normalize metrics values for parallel coordinates colors.
        const colors = normalizeRange(trialMetrics, trialMetricRange);
        const metricRange = getNumericRange(trialMetrics);

        setChartData(() => ({
          colors,
          data,
          metricRange,
          metricValues: trialMetrics,
          trialIds,
        }));
        setHasLoaded(true);
      },
    ).catch(e => setPageError(e));

    return () => canceler.abort();
  }, [ experiment.id, selectedBatch, selectedMetric ]);

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
    <div className={css.base}>
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
            value={hpList}
            onChange={handleHpChange}>
            {fullHpList.map(hpKey => <Option key={hpKey} value={hpKey}>{hpKey}</Option>)}
          </MultiSelect>
        </ResponsiveFilters>}
        title="HP Parallel Coordinates">
        <div className={css.container} onClick={handleChartClick}>
          {!hasLoaded || !chartData ? <Spinner /> : (
            <ParallelCoordinates
              colors={chartData.colors}
              data={chartData.data}
              dimensions={dimensions}
              smallerIsBetter={smallerIsBetter}
              onHover={handleChartHover}
              onUnhover={handleChartUnhover} />
          )}
          <div className={css.tooltip} ref={tooltipRef}>
            <div className={css.box}>
              <div className={css.row}>
                <div>Trial Id:</div>
                <div ref={trialIdRef} />
              </div>
              <div className={css.row}>
                <div>Metric:</div>
                <div ref={metricValueRef} />
              </div>
            </div>
          </div>
        </div>
      </Section>
    </div>
  );
};

export default HpParallelCoordinates;
