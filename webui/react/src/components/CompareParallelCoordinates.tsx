import Hermes, { DimensionType } from 'hermes-parallel-coordinates';
import Message from 'hew/Message';
import Spinner from 'hew/Spinner';
import { Title } from 'hew/Typography';
import React, { useEffect, useMemo, useState } from 'react';

import ParallelCoordinates from 'components/ParallelCoordinates';
import { useGlasbey } from 'hooks/useGlasbey';
import { TrialMetricData } from 'pages/TrialDetails/useTrialMetrics';
import {
  ExperimentWithTrial,
  HpTrialData,
  Hyperparameter,
  HyperparameterType,
  Primitive,
  Range,
  Scale,
  TrialItem,
  XAxisDomain,
} from 'types';
import { defaultNumericRange, getNumericRange, updateRange } from 'utils/chart';
import { flattenObject, isPrimitive } from 'utils/data';
import { metricToKey, metricToStr } from 'utils/metric';
import { numericSorter } from 'utils/sort';

import { CompareHyperparametersSettings } from './CompareHyperparameters.settings';
import css from './HpParallelCoordinates.module.scss';

interface Props {
  projectId: number;
  selectedExperiments: ExperimentWithTrial[];
  trials: TrialItem[];
  metricData: TrialMetricData;
  settings: CompareHyperparametersSettings;
  fullHParams: string[];
}

const CompareParallelCoordinates: React.FC<Props> = ({
  selectedExperiments,
  trials,
  settings,
  metricData,
  fullHParams,
}: Props) => {
  const [chartData, setChartData] = useState<HpTrialData | undefined>();
  const [hermesCreatedFilters, setHermesCreatedFilters] = useState<Hermes.Filters>({});

  const { data, isLoaded, setScale } = metricData;

  const colorMap = useGlasbey(selectedExperiments.map((e) => e.experiment.id));
  const selectedScale = settings.scale;

  useEffect(() => {
    setScale(selectedScale);
  }, [selectedScale, setScale]);

  const selectedMetric = settings.metric;
  const selectedHParams = settings.hParams;

  const experimentHyperparameters = useMemo(() => {
    const hpMap: Record<string, Hyperparameter> = {};
    selectedExperiments.forEach((exp) => {
      const hps = Object.keys(exp.experiment.hyperparameters);
      hps.forEach((hp) => (hpMap[hp] = exp.experiment.hyperparameters[hp]));
    });
    return hpMap;
  }, [selectedExperiments]);

  const config: Hermes.RecursivePartial<Hermes.Config> = useMemo(
    () => ({
      filters: hermesCreatedFilters,
      hooks: {
        onFilterChange: (filters: Hermes.Filters) => {
          setHermesCreatedFilters({ ...filters });
        },
        onReset: () => setHermesCreatedFilters({}),
      },
      style: {
        axes: { label: { placement: 'after' } },
        data: {
          series: chartData?.trialIds.map((trial) => ({
            lineWidth: 1,
            strokeStyle: colorMap[trial],
          })),
          targetDimensionKey: selectedMetric ? metricToStr(selectedMetric) : '',
        },
        dimension: { label: { angle: Math.PI / 4, truncate: 24 } },
        padding: [4, 120, 4, 16],
      },
    }),
    [colorMap, hermesCreatedFilters, selectedMetric, chartData?.trialIds],
  );

  const dimensions = useMemo(() => {
    const newDimensions: Hermes.Dimension[] = selectedHParams.map((key) => {
      const hp = experimentHyperparameters[key] || {};

      if (hp.type === HyperparameterType.Categorical || hp.vals) {
        return {
          categories: hp.vals?.map((val) => (isPrimitive(val) ? val : JSON.stringify(val))) ?? [],
          key,
          label: key,
          type: DimensionType.Categorical,
        };
      } else if (hp.type === HyperparameterType.Log) {
        return { key, label: key, logBase: hp.base, type: DimensionType.Logarithmic };
      }

      return { key, label: key, type: DimensionType.Linear };
    });

    if (chartData?.metricRange && selectedMetric) {
      const key = metricToStr(selectedMetric);
      newDimensions.push(
        selectedScale === Scale.Log
          ? {
            key,
            label: key,
            logBase: 10,
            type: DimensionType.Logarithmic,
          }
          : {
            key,
            label: key,
            type: DimensionType.Linear,
          },
      );
    }

    return newDimensions;
  }, [
    chartData?.metricRange,
    experimentHyperparameters,
    selectedMetric,
    selectedScale,
    selectedHParams,
  ]);

  useEffect(() => {
    if (!selectedMetric) return;
    const trialMetricsMap: Record<number, number> = {};
    const trialHpMap: Record<string, Record<number, Primitive>> = {};

    const trialHpdata: Record<string, Primitive[]> = {};
    let trialMetricRange: Range<number> = defaultNumericRange(true);

    trials?.forEach((trial) => {
      const expId = trial.experimentId;
      const key = metricToKey(selectedMetric);

      // Choose the final metric value for each trial
      const metricValue = data?.[trial.id]?.[key]?.data?.[XAxisDomain.Batches]?.at(-1)?.[1];

      if (!metricValue) return;
      trialMetricsMap[expId] = metricValue;

      trialMetricRange = updateRange<number>(trialMetricRange, metricValue);
      const flatHParams = {
        ...trial?.hyperparameters,
        ...flattenObject(trial?.hyperparameters || {}),
      };

      Object.keys(flatHParams).forEach((hpKey) => {
        const hpValue = flatHParams[hpKey];
        trialHpMap[hpKey] = trialHpMap[hpKey] ?? {};
        trialHpMap[hpKey][expId] = isPrimitive(hpValue)
          ? (hpValue as Primitive)
          : JSON.stringify(hpValue);
      });
    });

    const trialIds = Object.keys(trialMetricsMap)
      .map((id) => parseInt(id))
      .sort(numericSorter);

    Object.keys(trialHpMap).forEach((hpKey) => {
      trialHpdata[hpKey] = trialIds.map((trialId) => trialHpMap[hpKey][trialId]);
    });

    const metricKey = metricToStr(selectedMetric);
    const metricValues = trialIds.map((id) => trialMetricsMap[id]);
    trialHpdata[metricKey] = metricValues;

    const metricRange = getNumericRange(metricValues);
    setChartData({
      data: trialHpdata,
      metricRange,
      metricValues,
      trialIds,
    });
  }, [selectedExperiments, selectedMetric, fullHParams, metricData, selectedScale, trials, data]);

  if (!isLoaded) {
    return <Spinner center spinning />;
  }

  if (trials.length === 0) {
    return <Message title="No data available." />;
  }

  return (
    <div className={css.container}>
      <Title>Parallel Coordinates</Title>
      {chartData ? (
        <div className={css.chart}>
          <ParallelCoordinates
            config={config}
            data={chartData?.data ?? {}}
            dimensions={dimensions}
          />
        </div>
      ) : (
        <Message icon="warning" title="No data to plot." />
      )}
    </div>
  );
};

export default CompareParallelCoordinates;
