import Alert from 'hew/Alert';
import Divider from 'hew/Divider';
import Message from 'hew/Message';
import Spinner from 'hew/Spinner';
import { Title } from 'hew/Typography';
import React, { useCallback, useEffect, useMemo } from 'react';

import Section from 'components/Section';
import { useSettings } from 'hooks/useSettings';
import { ExperimentVisualizationType } from 'pages/ExperimentDetails/ExperimentVisualization';
import ExperimentVisualizationFilters, {
  VisualizationFilters,
} from 'pages/ExperimentDetails/ExperimentVisualization/ExperimentVisualizationFilters';
import { TrialMetricData } from 'pages/TrialDetails/useTrialMetrics';
import { ExperimentWithTrial, TrialItem } from 'types';

import CompareHeatMaps from './CompareHeatMaps';
import CompareParallelCoordinates from './CompareParallelCoordinates';
import {
  ExperimentHyperparametersSettings,
  settingsConfigForExperimentHyperparameters,
} from './CompareParallelCoordinates.settings';
import CompareScatterPlots from './CompareScatterPlots';
import css from './HpParallelCoordinates.module.scss';

interface Props {
  projectId: number;
  selectedExperiments: ExperimentWithTrial[];
  trials: TrialItem[];
  metricData: TrialMetricData;
}

const CompareHyperparameters: React.FC<Props> = ({
  selectedExperiments,
  trials,
  projectId,
  metricData,
}: Props) => {
  const { metrics, isLoaded, setScale } = metricData;

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
      (metric) =>
        metric.name === settings?.metric?.name && metric.group === settings?.metric?.group,
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

  if (!isLoaded) {
    return <Spinner center spinning />;
  }

  if (trials.length === 0) {
    return <Message title="No data available." />;
  }

  if (selectedExperiments.length !== 0 && metrics.length === 0) {
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
            <>
              <Title>Parallel Coordinates</Title>
              <CompareParallelCoordinates
                fullHParams={fullHParams}
                metricData={metricData}
                projectId={projectId}
                selectedExperiments={selectedExperiments}
                settings={settings}
                trials={trials}
              />
              <Divider />
              <Title>Scatter Plots</Title>
              <CompareScatterPlots
                fullHParams={fullHParams}
                metricData={metricData}
                selectedExperiments={selectedExperiments}
                settings={settings}
                trials={trials}
              />
              <Divider />
              <Title>Heat Map</Title>
              <CompareHeatMaps
                fullHParams={fullHParams}
                metricData={metricData}
                selectedExperiments={selectedExperiments}
                settings={settings}
                trials={trials}
              />
            </>
          )}
        </div>
      </div>
    </Section>
  );
};

export default CompareHyperparameters;
