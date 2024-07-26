import Alert from 'hew/Alert';
import Divider from 'hew/Divider';
import Message from 'hew/Message';
import Spinner from 'hew/Spinner';
import { Title } from 'hew/Typography';
import React, { useCallback, useEffect, useMemo } from 'react';

import Section from 'components/Section';
import useFeature from 'hooks/useFeature';
import { MapOfIdsToColors } from 'hooks/useGlasbey';
import { RunMetricData } from 'hooks/useMetrics';
import { useSettings } from 'hooks/useSettings';
import { ExperimentVisualizationType } from 'pages/ExperimentDetails/ExperimentVisualization';
import ExperimentVisualizationFilters, {
  VisualizationFilters,
} from 'pages/ExperimentDetails/ExperimentVisualization/ExperimentVisualizationFilters';
import { ExperimentWithTrial, FlatRun, TrialItem, XOR } from 'types';

import CompareHeatMaps from './CompareHeatMaps';
import {
  CompareHyperparametersSettings,
  settingsConfigForCompareHyperparameters,
} from './CompareHyperparameters.settings';
import CompareParallelCoordinates from './CompareParallelCoordinates';
import CompareScatterPlots from './CompareScatterPlots';
import css from './HpParallelCoordinates.module.scss';

interface BaseProps {
  colorMap: MapOfIdsToColors;
  projectId: number;
  metricData: RunMetricData;
}

type Props = XOR<
  { selectedExperiments: ExperimentWithTrial[]; trials: TrialItem[] },
  { selectedRuns: FlatRun[] }
> &
  BaseProps;

export const NO_DATA_MESSAGE = 'No data available.';

const CompareHyperparameters: React.FC<Props> = ({
  selectedExperiments,
  selectedRuns,
  colorMap,
  trials,
  projectId,
  metricData,
}: Props) => {
  const { metrics, isLoaded: metricsLoaded, setScale } = metricData;

  const fullHParams: string[] = useMemo(() => {
    const hpParams = new Set<string>();
    trials?.forEach((trial) =>
      Object.keys(trial.hyperparameters).forEach((hp) => hpParams.add(hp)),
    );
    selectedRuns?.forEach((run) =>
      Object.keys(run.hyperparameters ?? {}).forEach((hp) => hpParams.add(hp)),
    );
    return Array.from(hpParams);
  }, [selectedRuns, trials]);

  const settingsConfig = useMemo(
    () => settingsConfigForCompareHyperparameters(fullHParams, projectId),
    [fullHParams, projectId],
  );
  const f_flat_runs = useFeature().isOn('flat_runs');

  const {
    settings,
    isLoading: isLoadingSettings,
    updateSettings,
    resetSettings,
  } = useSettings<CompareHyperparametersSettings>(settingsConfig);

  useEffect(() => {
    setScale(settings.scale);
  }, [settings.scale, setScale]);

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
    if (metrics.length === 0) return;
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

  if (!metricsLoaded || isLoadingSettings) {
    return <Spinner center spinning />;
  }

  if ((trials ?? selectedRuns).length === 0) {
    return <Message title={NO_DATA_MESSAGE} />;
  }

  if ((selectedExperiments ?? selectedRuns).length !== 0 && metrics.length === 0) {
    const entityName = f_flat_runs ? 'searches' : 'experiments';
    return (
      <div className={css.waiting}>
        <Alert
          description={`Please wait until the ${entityName} are further along.`}
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
          {(selectedExperiments ?? selectedRuns).length > 0 && (
            <>
              <Title>Parallel Coordinates</Title>
              {selectedRuns ? (
                <CompareParallelCoordinates
                  colorMap={colorMap}
                  fullHParams={fullHParams}
                  metricData={metricData}
                  projectId={projectId}
                  selectedRuns={selectedRuns}
                  settings={settings}
                />
              ) : (
                <CompareParallelCoordinates
                  colorMap={colorMap}
                  fullHParams={fullHParams}
                  metricData={metricData}
                  projectId={projectId}
                  selectedExperiments={selectedExperiments}
                  settings={settings}
                  trials={trials}
                />
              )}
              <Divider />
              <Title>Scatter Plots</Title>
              {selectedRuns ? (
                <CompareScatterPlots
                  fullHParams={fullHParams}
                  metricData={metricData}
                  selectedRuns={selectedRuns}
                  settings={settings}
                />
              ) : (
                <CompareScatterPlots
                  fullHParams={fullHParams}
                  metricData={metricData}
                  selectedExperiments={selectedExperiments}
                  settings={settings}
                  trials={trials}
                />
              )}
              <Divider />
              <Title>Heat Maps</Title>
              {selectedRuns ? (
                <CompareHeatMaps
                  fullHParams={fullHParams}
                  metricData={metricData}
                  selectedRuns={selectedRuns}
                  settings={settings}
                />
              ) : (
                <CompareHeatMaps
                  fullHParams={fullHParams}
                  metricData={metricData}
                  selectedExperiments={selectedExperiments}
                  settings={settings}
                  trials={trials}
                />
              )}
            </>
          )}
        </div>
      </div>
    </Section>
  );
};

export default CompareHyperparameters;
