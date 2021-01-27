import { Alert, Select } from 'antd';
import { SelectValue } from 'antd/lib/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import MetricSelectFilter from 'components/MetricSelectFilter';
import ParallelCoordinates, {
  ConfigDimension, dimensionTypeMap, Range,
} from 'components/ParallelCoordinates';
import ResponsiveFilters from 'components/ResponsiveFilters';
import Section from 'components/Section';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import { V1TrialsSnapshotResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import {
  ExperimentBase, ExperimentHyperParamType, ExperimentHyperParamValue, MetricName, metricTypeParamMap,
} from 'types';
import { hpSorter, isObject } from 'utils/data';

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
  hparams: Record<string, ExperimentHyperParamValue[]>;
  lineIds: number[];
}

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
  const [ data, setData ] = useState<HpTrialData>({ colors: [], hparams: {}, lineIds: [] });

  const dimensions = useMemo(() => {
    return Object.keys(experiment.config.hyperparameters).map(key => {
      const hp = experiment.config.hyperparameters[key];
      const isCategorical = hp.type === ExperimentHyperParamType.Categorical;
      const range = hp.minval != null && hp.maxval != null ?
        [ hp.minval, hp.maxval ] as Range : undefined;
      const dimension: ConfigDimension = { label: key, range, type: dimensionTypeMap[hp.type] };

      if (isCategorical && Array.isArray(hp.vals)) {
        dimension.categories = hp.vals.map(val => val.toString());
      }

      return dimension;
    });
  }, [ experiment.config.hyperparameters ]);

  const resetData = useCallback(() => {
    // setChartData([]);
    // setTrialHps([]);
    // setTrialIds([]);
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
        if (!event || !event.trials || !isObject(event.trials)) return;

        const trialIds: number[] = [];
        const trialMetrics: number[] = [];
        const trialHps: Record<string, ExperimentHyperParamValue[]> = {};
        const trialHpMap: Record<string, Record<number, ExperimentHyperParamValue>> = {};
        const trialHpRanges: Record<string, Range> = {};

        event.trials.forEach(trial => {
          trialIds.push(trial.trialId);
          trialMetrics.push(trial.metric);

          Object.keys(trial.hparams || {}).forEach(hpKey => {
            const hpValue = trial.hparams[hpKey];
            const defaultRange = [ Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY ];
            trialHpMap[hpKey] = trialHpMap[hpKey] || {};
            trialHpMap[hpKey][trial.trialId] = hpValue;
            trialHpRanges[hpKey] = trialHpRanges[hpKey] || defaultRange;

            if (hpSorter(trialHpRanges[hpKey][0], hpValue) === -1) {
              trialHpRanges[hpKey][0] = hpValue;
            }
            if (hpSorter(trialHpRanges[hpKey][1], hpValue) === 1) {
              trialHpRanges[hpKey][1] = hpValue;
            }
          });
        });

        Object.keys(trialHpMap).forEach(hpKey => {
          trialHps[hpKey] = trialIds.map(trialId => trialHpMap[hpKey][trialId]);
        });

        setData({
          colors: trialMetrics,
          hparams: trialHps,
          lineIds: trialIds,
        });
        console.log(trialIds, trialMetrics, trialHps, trialHpRanges);

        setHasLoaded(true);
      },
    );

    return () => canceler.abort();
  }, [ experiment.id, selectedBatch, selectedMetric ]);

  if (!hasLoaded) return <Spinner />;

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
        </ResponsiveFilters>}
        title="HP Parallel Coordinates">
        <div className={css.container}>
          <ParallelCoordinates
            colors={data.colors}
            data={data.hparams}
            dimensions={dimensions}
            lineIds={data.lineIds} />
        </div>
      </Section>
    </div>
  );
};

export default HpParallelCoordinates;
