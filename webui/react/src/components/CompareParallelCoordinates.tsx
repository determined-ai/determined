import { Alert } from 'antd';
import Hermes, { DimensionType } from 'hermes-parallel-coordinates';
import Message from 'hew/Message';
import Spinner from 'hew/Spinner';
import React, { useEffect, useMemo, useState } from 'react';

import ParallelCoordinates from 'components/ParallelCoordinates';
import { useGlasbey } from 'hooks/useGlasbey';
import { TrialMetricData } from 'pages/TrialDetails/useTrialMetrics';
import {
  ExperimentWithTrial,
  FlatRun,
  HpTrialData,
  Hyperparameter,
  HyperparameterType,
  Primitive,
  Scale,
  TrialItem,
  XAxisDomain,
  XOR,
} from 'types';
import { getNumericRange } from 'utils/chart';
import { flattenObject, isPrimitive } from 'utils/data';
import { metricToKey, metricToStr } from 'utils/metric';
import { numericSorter } from 'utils/sort';

import { CompareHyperparametersSettings } from './CompareHyperparameters.settings';
import css from './HpParallelCoordinates.module.scss';

export const COMPARE_PARALLEL_COORDINATES = 'compare-parallel-coordinates';

interface BaseProps {
  projectId: number;
  metricData: TrialMetricData;
  settings: CompareHyperparametersSettings;
  fullHParams: string[];
}

type Props = XOR<
  { selectedExperiments: ExperimentWithTrial[]; trials: TrialItem[] },
  { selectedRuns: FlatRun[] }
> &
  BaseProps;

const CompareParallelCoordinates: React.FC<Props> = ({
  selectedExperiments,
  trials,
  settings,
  metricData,
  fullHParams,
  selectedRuns,
}: Props) => {
  const [chartData, setChartData] = useState<HpTrialData | undefined>();
  const [hermesCreatedFilters, setHermesCreatedFilters] = useState<Hermes.Filters>({});

  const { metrics, data, isLoaded, setScale } = metricData;

  const colorMap = useGlasbey(
    selectedExperiments
      ? selectedExperiments.map((e) => e.experiment.id)
      : selectedRuns.map((r) => r.id),
  );
  const selectedScale = settings.scale;

  useEffect(() => {
    setScale(selectedScale);
  }, [selectedScale, setScale]);

  const selectedMetric = settings.metric;
  const selectedHParams = settings.hParams;

  const experimentHyperparameters = useMemo(() => {
    const hpMap: Record<string, Hyperparameter> = {};
    selectedExperiments?.forEach((exp) => {
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
    const metricsMap: Record<number, number> = {};
    const hpMap: Record<string, Record<number, Primitive>> = {};

    const hpData: Record<string, Primitive[]> = {};

    if (trials) {
      trials.forEach((trial) => {
        const expId = trial.experimentId;
        const key = metricToKey(selectedMetric);

        // Choose the final metric value for each trial
        const metricValue = data?.[trial.id]?.[key]?.data?.[XAxisDomain.Batches]?.at(-1)?.[1];

        if (!metricValue) return;
        metricsMap[expId] = metricValue;

        const flatHParams = {
          ...trial?.hyperparameters,
          ...flattenObject(trial?.hyperparameters || {}),
        };

        Object.keys(flatHParams).forEach((hpKey) => {
          const hpValue = flatHParams[hpKey];
          hpMap[hpKey] = hpMap[hpKey] ?? {};
          hpMap[hpKey][expId] = isPrimitive(hpValue)
            ? (hpValue as Primitive)
            : JSON.stringify(hpValue);
        });
      });
    } else if (selectedRuns) {
      selectedRuns.forEach((run) => {
        const key = metricToKey(selectedMetric);

        // Choose the final metric value for each trial
        const metricValue = data?.[run.id]?.[key]?.data?.[XAxisDomain.Batches]?.at(-1)?.[1];

        if (!metricValue) return;
        metricsMap[run.id] = metricValue;

        const flatHParams = {
          ...run.hyperparameters,
          ...flattenObject(run.hyperparameters || {}),
        };

        Object.keys(flatHParams).forEach((hpKey) => {
          const hpValue = flatHParams[hpKey];
          hpMap[hpKey] = hpMap[hpKey] ?? {};
          hpMap[hpKey][run.id] = isPrimitive(hpValue)
            ? (hpValue as Primitive)
            : JSON.stringify(hpValue);
        });
      });
    }

    const trialIds = Object.keys(metricsMap)
      .map((id) => parseInt(id))
      .sort(numericSorter);

    Object.keys(hpMap).forEach((hpKey) => {
      hpData[hpKey] = trialIds.map((trialId) => hpMap[hpKey][trialId]);
    });

    const metricKey = metricToStr(selectedMetric);
    const metricValues = trialIds.map((id) => metricsMap[id]);
    hpData[metricKey] = metricValues;

    const metricRange = getNumericRange(metricValues);
    setChartData({
      data: hpData,
      metricRange,
      metricValues,
      trialIds,
    });
  }, [
    selectedExperiments,
    selectedMetric,
    fullHParams,
    metricData,
    selectedScale,
    trials,
    data,
    selectedRuns,
  ]);

  if (!isLoaded) {
    return <Spinner center spinning />;
  }

  if ((trials ?? selectedRuns).length === 0) {
    return <Message title="No data available." />;
  }

  if (!chartData || ((selectedExperiments ?? selectedRuns).length !== 0 && metrics.length === 0)) {
    return (
      <div className={css.waiting}>
        <Alert
          description="Please wait until the experiments are further along."
          message="Not enough data points to plot."
        />
        <Spinner center spinning />
      </div>
    );
  }

  return (
    <div className={css.container}>
      {(selectedExperiments ?? selectedRuns).length > 0 && (
        <div className={css.chart} data-testid={COMPARE_PARALLEL_COORDINATES}>
          <ParallelCoordinates
            config={config}
            data={chartData?.data ?? {}}
            dimensions={dimensions}
          />
        </div>
      )}
    </div>
  );
};

export default CompareParallelCoordinates;
