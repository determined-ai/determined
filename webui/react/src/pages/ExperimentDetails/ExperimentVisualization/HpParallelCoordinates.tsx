import { Alert, Select } from 'antd';
import { SelectValue } from 'antd/lib/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Message, { MessageType } from 'components/Message';
import MetricSelectFilter from 'components/MetricSelectFilter';
import MultiSelect from 'components/MultiSelect';
import ParallelCoordinates, { Dimension, dimensionTypeMap } from 'components/ParallelCoordinates';
import ResponsiveFilters from 'components/ResponsiveFilters';
import Section from 'components/Section';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import { V1TrialsSnapshotResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import {
  ExperimentBase, ExperimentHyperParamType, MetricName, metricTypeParamMap, Primitive, Range, RunState,
} from 'types';
import { defaultNumericRange, normalizeRange, updateRange } from 'utils/chart';
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
  hparams: Record<string, Primitive[]>;
  lineIds: number[];
}

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
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ chartData, setChartData ] = useState<HpTrialData>();
  const [ pageError, setPageError ] = useState<Error>();
  const fullHpList = Object.keys(experiment.config.hyperparameters) || [];
  const limitedHpList = fullHpList.slice(0, MAX_HP_COUNT);
  const defaultHpList = storage.get<string[]>(STORAGE_HP_KEY);
  const [ hpList, setHpList ] = useState<string[]>(defaultHpList || limitedHpList);

  const isExperimentTerminal = terminalRunStates.has(experiment.state as RunState);

  const dimensions = useMemo(() => {
    return hpList.map(key => {
      const hp = experiment.config.hyperparameters[key];
      const isConstant = hp.type === ExperimentHyperParamType.Constant;
      const dimension: Dimension = {
        categories: hp.vals,
        label: key,
        type: dimensionTypeMap[hp.type],
      };

      if (isConstant && hp.val != null) {
        dimension.range = updateRange(undefined, hp.val);
      } else if (hp.minval != null && hp.maxval != null) {
        dimension.range = [ hp.minval, hp.maxval ] as Range<number>;
      }

      return dimension;
    });
  }, [ experiment.config.hyperparameters, hpList ]);

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
        const trialHps: Record<string, Primitive[]> = {};
        const trialHpMap: Record<string, Record<number, Primitive>> = {};
        const trialHpRanges: Record<string, Range> = {};
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
          trialHps[hpKey] = trialIds.map(trialId => trialHpMap[hpKey][trialId]);
        });

        // Normalize metrics values for parallel coordinates colors.
        const colors = normalizeRange(trialMetrics, trialMetricRange);

        setChartData({ colors, hparams: trialHps, lineIds: trialIds });
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
            <ParallelCoordinates
              colors={chartData.colors}
              data={chartData.hparams}
              dimensions={dimensions}
              lineIds={chartData.lineIds} />
          )}
        </div>
      </Section>
    </div>
  );
};

export default HpParallelCoordinates;
