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
import themes, { defaultThemeId } from 'themes';
import {
  ExperimentBase, ExperimentHyperParamType, MetricName, MetricType, metricTypeParamMap,
  Primitive, Range, RunState,
} from 'types';
import { defaultNumericRange, getNumericRange, updateRange } from 'utils/chart';
import { ColorScale } from 'utils/color';
import { clone, isNumber, numericSorter } from 'utils/data';
import { metricNameToStr } from 'utils/string';
import { terminalRunStates } from 'utils/types';

import css from './HpParallelCoordinates.module.scss';
import HpTrialTable, { TrialHParams } from './HpTrialTable';

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
  data: Record<string, Primitive[]>;
  metricRange?: Range<number>;
  metricValues: number[];
  trialIds: number[];
}

const STORAGE_PATH = 'experiment-visualization';
const STORAGE_HP_KEY = 'hps';
const MAX_HP_COUNT = 10;
const DEFAULT_SCALE_COLORS: Range<string> = [
  themes[defaultThemeId].colors.danger.light,
  themes[defaultThemeId].colors.action.normal,
];
const REVERSE_SCALE_COLORS = clone(DEFAULT_SCALE_COLORS).reverse();
const NEUTRAL_SCALE_COLORS: Range<string> = [
  'rgb(255, 207, 0)',
  themes[defaultThemeId].colors.action.normal,
];

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
  const [ trialHps, setTrialHps ] = useState<TrialHParams[]>([]);
  const [ pageError, setPageError ] = useState<Error>();
  const fullHpList = Object.keys(experiment.config.hyperparameters) || [];
  const limitedHpList = fullHpList.slice(0, MAX_HP_COUNT);
  const defaultHpList = storage.get<string[]>(STORAGE_HP_KEY);
  const [ hpList, setHpList ] = useState<string[]>(defaultHpList || limitedHpList);
  const [ filteredTrialIdMap, setFilteredTrialIdMap ] = useState<Record<number, boolean>>();

  const isExperimentTerminal = terminalRunStates.has(experiment.state as RunState);

  const smallerIsBetter = useMemo(() => {
    if (selectedMetric.type === MetricType.Validation &&
        selectedMetric.name === experiment.config.searcher.metric) {
      return experiment.config.searcher.smallerIsBetter;
    }
    return undefined;
  }, [ experiment.config.searcher, selectedMetric ]);

  const colorScale: ColorScale[] = useMemo(() => {
    let colors = NEUTRAL_SCALE_COLORS;
    if (smallerIsBetter != null) {
      colors = smallerIsBetter ? REVERSE_SCALE_COLORS : DEFAULT_SCALE_COLORS;
    }
    return colors.map((color, index) => {
      if (chartData?.metricRange) {
        const scale = chartData?.metricRange ? chartData?.metricRange[index] : index;
        return { color, scale };
      }
      return { color, scale: index };
    });
  }, [ chartData?.metricRange, smallerIsBetter ]);

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
        label: metricNameToStr(selectedMetric),
        range: chartData.metricRange,
        type: DimensionType.Scalar,
      });
    }

    return newDimensions;
  }, [ chartData, experiment.config.hyperparameters, hpList, selectedMetric ]);

  const resetData = useCallback(() => {
    setChartData(undefined);
    setTrialHps([]);
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

  const handleChartFilter = useCallback((constraints: Record<string, Range>) => {
    if (!chartData) return;

    // Figure out which trials fit within the user provided constraints.
    const newFilteredTrialIdMap = chartData.trialIds.reduce((acc, trialId) => {
      acc[trialId] = true;
      return acc;
    }, {} as Record<number, boolean>);

    Object.entries(constraints).forEach(([ key, range ]) => {
      if (!isNumber(range[0]) || !isNumber(range[1])) return;
      if (!chartData.data[key]) return;

      const values = chartData.data[key];
      values.forEach((value, index) => {
        if (value >= range[0] && value <= range[1]) return;
        const trialId = chartData.trialIds[index];
        newFilteredTrialIdMap[trialId] = false;
      });
    });

    setFilteredTrialIdMap(newFilteredTrialIdMap);
  }, [ chartData ]);

  const handleTableClick = useCallback((event: React.MouseEvent, record: TrialHParams) => {
    if (record.id) handlePath(event, { path: paths.trialDetails(record.id, experiment.id) });
  }, [ experiment.id ]);

  useEffect(() => {
    const canceler = new AbortController();
    const trialMetricsMap: Record<number, number> = {};
    const trialHpTableMap: Record<number, TrialHParams> = {};
    const trialHpMap: Record<string, Record<number, Primitive>> = {};

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
        if (!event || !event.trials || !Array.isArray(event.trials)) return;

        const data: Record<string, Primitive[]> = {};
        let trialMetricRange: Range<number> = defaultNumericRange(true);

        event.trials.forEach(trial => {
          const id = trial.trialId;
          trialMetricsMap[id] = trial.metric;
          trialMetricRange = updateRange<number>(trialMetricRange, trial.metric);

          Object.keys(trial.hparams || {}).forEach(hpKey => {
            const hpValue = trial.hparams[hpKey];
            trialHpMap[hpKey] = trialHpMap[hpKey] || {};
            trialHpMap[hpKey][id] = hpValue;
          });

          trialHpTableMap[id] = {
            hparams: clone(trial.hparams),
            id,
            metric: trial.metric,
          };
        });

        const trialIds = Object.keys(trialMetricsMap)
          .map(id => parseInt(id))
          .sort(numericSorter);

        Object.keys(trialHpMap).forEach(hpKey => {
          data[hpKey] = trialIds.map(trialId => trialHpMap[hpKey][trialId]);
        });

        // Add metric of interest
        const metricKey = metricNameToStr(selectedMetric);
        const metricValues = trialIds.map(id => trialMetricsMap[id]);
        data[metricKey] = metricValues;

        // Normalize metrics values for parallel coordinates colors.
        const metricRange = getNumericRange(metricValues);

        // Gather hparams for trial table
        const newTrialHps = trialIds.map(id => trialHpTableMap[id]);
        setTrialHps(newTrialHps);

        setChartData({
          data,
          metricRange,
          metricValues,
          trialIds,
        });
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
        <div className={css.container}>
          {!hasLoaded || !chartData ? <Spinner /> : (
            <>
              <div className={css.chart}>
                <ParallelCoordinates
                  colorScale={colorScale}
                  colorScaleKey={metricNameToStr(selectedMetric)}
                  data={chartData.data}
                  dimensions={dimensions}
                  onFilter={handleChartFilter}
                />
              </div>
              <div className={css.table}>
                <HpTrialTable
                  colorScale={colorScale}
                  filteredTrialIdMap={filteredTrialIdMap}
                  hyperparameters={experiment.config.hyperparameters || {}}
                  metric={selectedMetric}
                  trialHps={trialHps}
                  trialIds={chartData.trialIds}
                  onClick={handleTableClick}
                />
              </div>
            </>
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
