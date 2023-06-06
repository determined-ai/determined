import { Alert } from 'antd';
import Hermes, { DimensionType } from 'hermes-parallel-coordinates';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import ParallelCoordinates from 'components/ParallelCoordinates';
import Section from 'components/Section';
import { useSettings } from 'hooks/useSettings';
import { ExperimentVisualizationType } from 'pages/ExperimentDetails/ExperimentVisualization';
import ExperimentVisualizationFilters, {
  VisualizationFilters,
} from 'pages/ExperimentDetails/ExperimentVisualization/ExperimentVisualizationFilters';
import { useTrialMetrics } from 'pages/TrialDetails/useTrialMetrics';
import Spinner from 'shared/components/Spinner/Spinner';
import { Primitive, Range } from 'shared/types';
import { flattenObject, isPrimitive } from 'shared/utils/data';
import { isEqual } from 'shared/utils/data';
import { numericSorter } from 'shared/utils/sort';
import {
  ExperimentWithTrial,
  Hyperparameter,
  HyperparameterType,
  MetricType,
  Scale,
  TrialItem,
} from 'types';
import { defaultNumericRange, getNumericRange, updateRange } from 'utils/chart';
import { metricToStr } from 'utils/metric';

import {
  ExperimentHyperparametersSettings,
  settingsConfigForExperimentHyperparameters,
} from './CompareParallelCoordinates.settings';
import css from './HpParallelCoordinates.module.scss';
import { useGlasbey } from './useGlasbey';

interface Props {
  selectedExperiments: ExperimentWithTrial[];
}

interface HpTrialData {
  data: Record<string, Primitive[]>;
  metricRange?: Range<number>;
  metricValues: number[];
  trialIds: number[];
}

const CompareParallelCoordinates: React.FC<Props> = ({ selectedExperiments }: Props) => {
  const [chartData, setChartData] = useState<HpTrialData>();
  const [hermesCreatedFilters, setHermesCreatedFilters] = useState<Hermes.Filters>({});
  const [trials, setTrials] = useState<TrialItem[]>([]);

  useEffect(() => {
    const ts: TrialItem[] = [];
    selectedExperiments.forEach((e) => e.bestTrial && ts.push(e.bestTrial));
    setTrials((prev: TrialItem[]) => {
      return isEqual(
        prev?.map((e) => e.id),
        ts?.map((e) => e?.id),
      )
        ? prev
        : ts;
    });
  }, [selectedExperiments]);

  const fullHParams: string[] = useMemo(() => {
    const hpParams = new Set<string>();
    selectedExperiments.forEach((exp) =>
      Object.keys(exp.experiment.hyperparameters).forEach((hp) => hpParams.add(hp)),
    );
    return Array.from(hpParams);
  }, [selectedExperiments]);

  const settingsConfig = useMemo(
    () => settingsConfigForExperimentHyperparameters(fullHParams),
    [fullHParams],
  );

  const { settings, updateSettings, resetSettings } =
    useSettings<ExperimentHyperparametersSettings>(settingsConfig);

  const { metrics, data, setScale } = useTrialMetrics(trials);

  const colorMap = useGlasbey(selectedExperiments.map((e) => e.experiment.id));

  const selectedScale = settings.scale;

  useEffect(() => {
    setScale(selectedScale);
  }, [selectedScale, setScale]);

  const filters: VisualizationFilters = useMemo(
    () => ({
      hParams: settings.hParams,
      metric: settings.metric,
      scale: settings.scale,
    }),
    [settings.hParams, settings.metric, settings.scale],
  );

  const handleFiltersChange = useCallback(
    (filters: Partial<VisualizationFilters>) => {
      updateSettings(filters);
    },
    [updateSettings],
  );

  const handleFiltersReset = useCallback(() => {
    resetSettings();
  }, [resetSettings]);

  useEffect(() => {
    if (settings.metric !== undefined) return;
    const activeMetricFound = metrics.find((metric) => metric.type === MetricType.Validation);
    updateSettings({ metric: activeMetricFound ?? metrics.first() });
  }, [selectedExperiments, metrics, settings.metric, updateSettings]);

  const visualizationFilters = useMemo(() => {
    return (
      <ExperimentVisualizationFilters
        filters={filters}
        fullHParams={fullHParams}
        metrics={metrics}
        type={ExperimentVisualizationType.HpParallelCoordinates}
        onChange={handleFiltersChange}
        onReset={handleFiltersReset}
      />
    );
  }, [fullHParams, handleFiltersChange, handleFiltersReset, metrics, filters]);

  const selectedMetric = settings.metric;
  const selectedHParams = settings.hParams;

  const hyperparameters = useMemo(() => {
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
      const hp = hyperparameters[key] || {};

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
  }, [chartData?.metricRange, hyperparameters, selectedMetric, selectedScale, selectedHParams]);

  useEffect(() => {
    if (!selectedMetric) return;
    const trialMetricsMap: Record<number, number> = {};
    const trialHpMap: Record<string, Record<number, Primitive>> = {};

    const tdata: Record<string, Primitive[]> = {};
    let trialMetricRange: Range<number> = defaultNumericRange(true);
    trials?.forEach((trial) => {
      const expId = trial.experimentId;
      const key = `${selectedMetric.type}|${selectedMetric.name}`;

      // Need to determine the correct metric value here since the table is no longer based on batch
      const metricValue = data?.[trial.id]?.[key]?.data?.[XAxisDomain.Batches]?.[0]?.[1];

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
      tdata[hpKey] = trialIds.map((trialId) => trialHpMap[hpKey][trialId]);
    });

    const metricKey = metricToStr(selectedMetric);
    const metricValues = trialIds.map((id) => trialMetricsMap[id]);
    tdata[metricKey] = metricValues;

    const metricRange = getNumericRange(metricValues);

    setChartData({
      data: tdata,
      metricRange,
      metricValues,
      trialIds,
    });
  }, [selectedExperiments, selectedMetric, fullHParams, data, selectedScale, metrics, trials]);

  if (!chartData) {
    return (
      <div className={css.waiting}>
        <Alert
          description="Please wait until the experiment is further along."
          message="Not enough data points to plot."
        />
        <Spinner />
      </div>
    );
  }

  return (
    <Section bodyBorder bodyScroll filters={visualizationFilters}>
      <div className={css.container}>
        <div className={css.chart}>
          {selectedExperiments.length > 0 && (
            <ParallelCoordinates
              config={config}
              data={chartData?.data ?? {}}
              dimensions={dimensions}
            />
          )}
        </div>
      </div>
    </Section>
  );
};

export default CompareParallelCoordinates;
