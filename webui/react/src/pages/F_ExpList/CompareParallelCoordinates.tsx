import { Alert } from 'antd';
import Hermes, { DimensionType } from 'hermes-parallel-coordinates';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Spinner from 'components/kit/internal/Spinner/Spinner';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import Message, { MessageType } from 'components/Message';
import ParallelCoordinates from 'components/ParallelCoordinates';
import Section from 'components/Section';
import { useGlasbey } from 'hooks/useGlasbey';
import { useSettings } from 'hooks/useSettings';
import { ExperimentVisualizationType } from 'pages/ExperimentDetails/ExperimentVisualization';
import ExperimentVisualizationFilters, {
  VisualizationFilters,
} from 'pages/ExperimentDetails/ExperimentVisualization/ExperimentVisualizationFilters';
import { TrialMetricData } from 'pages/TrialDetails/useTrialMetrics';
import { Primitive, Range } from 'types';
import {
  ExperimentWithTrial,
  HpTrialData,
  Hyperparameter,
  HyperparameterType,
  Scale,
  TrialItem,
} from 'types';
import { defaultNumericRange, getNumericRange, updateRange } from 'utils/chart';
import { flattenObject, isPrimitive } from 'utils/data';
import { metricToStr } from 'utils/metric';
import { numericSorter } from 'utils/sort';

import {
  ExperimentHyperparametersSettings,
  settingsConfigForExperimentHyperparameters,
} from './CompareParallelCoordinates.settings';
import css from './HpParallelCoordinates.module.scss';

interface Props {
  projectId: number;
  selectedExperiments: ExperimentWithTrial[];
  trials: TrialItem[];
  metricData: TrialMetricData;
}

const CompareParallelCoordinates: React.FC<Props> = ({
  selectedExperiments,
  trials,
  projectId,
  metricData,
}: Props) => {
  const [chartData, setChartData] = useState<HpTrialData | undefined>();
  const [hermesCreatedFilters, setHermesCreatedFilters] = useState<Hermes.Filters>({});

  const fullHParams: string[] = useMemo(() => {
    const hpParams = new Set<string>();
    trials.forEach((trial) => Object.keys(trial.hyperparameters).forEach((hp) => hpParams.add(hp)));
    return Array.from(hpParams);
  }, [trials]);

  const settingsConfig = useMemo(
    () => settingsConfigForExperimentHyperparameters(fullHParams, projectId),
    [fullHParams, projectId],
  );

  const { settings, updateSettings, resetSettings } =
    useSettings<ExperimentHyperparametersSettings>(settingsConfig);

  const { metrics, data, isLoaded, setScale } = metricData;

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
    const activeMetricFound = metrics.find(
      (metric) => metric.name === settings?.metric?.name && metric.type === settings?.metric?.type,
    );
    updateSettings({ metric: activeMetricFound ?? metrics.first() });
  }, [selectedExperiments, metrics, settings.metric, updateSettings]);

  useEffect(() => {
    if (settings.hParams !== undefined) {
      if (settings.hParams.length === 0 && fullHParams.length > 0) {
        updateSettings({ hParams: fullHParams.slice(0, 10) });
      } else {
        const activeHParams = settings.hParams.filter((hp) => fullHParams.includes(hp));
        updateSettings({ hParams: activeHParams });
      }
    } else {
      updateSettings({ hParams: fullHParams });
    }
  }, [selectedExperiments, fullHParams, settings.hParams, updateSettings]);

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
      const key = `${selectedMetric.type}|${selectedMetric.name}`;

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
    return <Message title="No data to plot." type={MessageType.Empty} />;
  }

  if (!chartData || (selectedExperiments.length !== 0 && metrics.length === 0)) {
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
