import { Alert } from 'antd';
import Hermes, { DimensionType } from 'hermes-parallel-coordinates';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Empty from 'components/kit/Empty';
import ParallelCoordinates from 'components/ParallelCoordinates';
import Section from 'components/Section';
import TableBatch from 'components/Table/TableBatch';
import { terminalRunStates } from 'constants/states';
import { openOrCreateTensorBoard } from 'services/api';
import { V1TrialsSnapshotResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import Spinner from 'shared/components/Spinner/Spinner';
import useUI from 'shared/contexts/stores/UI';
import { Primitive, Range, RawJson } from 'shared/types';
import { rgba2str, str2rgba } from 'shared/utils/color';
import { clone, flattenObject, isPrimitive } from 'shared/utils/data';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { numericSorter } from 'shared/utils/sort';
import {
  ExperimentAction as Action,
  CommandResponse,
  ExperimentWithTrial,
  Hyperparameter,
  HyperparameterType,
  Metric,
  MetricType,
  metricTypeParamMap,
  Scale,
  TrialDetails,
} from 'types';
import { defaultNumericRange, getColorScale, getNumericRange, updateRange } from 'utils/chart';
import { metricToStr } from 'utils/metric';
import { openCommandResponse } from 'utils/wait';
import { MapOfIdsToColors } from './useGlasbey';
import { Space } from 'antd';

import useMetricNames from 'hooks/useMetricNames';
import { useSettings } from 'hooks/useSettings';
import { ExperimentVisualizationType } from 'pages/ExperimentDetails/ExperimentVisualization';
import ExperimentVisualizationFilters, {
  VisualizationFilters,
} from 'pages/ExperimentDetails/ExperimentVisualization/ExperimentVisualizationFilters';
import { V1MetricBatchesResponse } from 'services/api-ts-sdk';
import { alphaNumericSorter } from 'shared/utils/sort';
import {
  ExperimentBase,
  ExperimentSearcherName,
} from 'types';
import handleError from 'utils/error';

import {
  ExperimentHyperparametersSettings,
  settingsConfigForExperimentHyperparameters,
} from './MultiTrialDetailsHyperparameters.settings'

import css from './HpParallelCoordinates.module.scss';
import HpTrialTable, { TrialHParams } from '../ExperimentDetails/ExperimentVisualization/HpTrialTable'
import experiments from 'stores/experiments';

interface Props {
  colorMap:MapOfIdsToColors;
  experiments: ExperimentWithTrial[];
  workspaceId: number;
}

interface HpTrialData {
  data: Record<string, Primitive[]>;
  metricRange?: Range<number>;
  metricValues: number[];
  trialIds: number[];
}

const HpParallelCoordinates: React.FC<Props> = ({
  colorMap,
  experiments,
  workspaceId
}: Props) => {
  const { ui } = useUI();
  const tooltipRef = useRef<HTMLDivElement>(null);
  const trialIdRef = useRef<HTMLDivElement>(null);
  const metricValueRef = useRef<HTMLDivElement>(null);
  const [hasLoaded, setHasLoaded] = useState(false);
  const [chartData, setChartData] = useState<HpTrialData>();
  const [trialHps, setTrialHps] = useState<TrialHParams[]>([]);
  const [pageError, setPageError] = useState<Error>();
  const [filteredTrialIdMap, setFilteredTrialIdMap] = useState<Record<number, boolean>>();
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [hermesCreatedFilters, setHermesCreatedFilters] = useState<Hermes.Filters>({});

  const fullHParams: string[] = useMemo(() => {
    const hpParams = new Set<string>();
    experiments.forEach((exp) => (Object.keys(exp.experiment.hyperparameters).forEach(hp => hpParams.add(hp))))
    return Array.from(hpParams)
  }, [experiments])
  
  const settingsConfig = useMemo(
    () => settingsConfigForExperimentHyperparameters(fullHParams),
    [fullHParams],
  );
  
  const { settings, updateSettings, resetSettings, activeSettings } =
    useSettings<ExperimentHyperparametersSettings>(settingsConfig);

  
  const filters: VisualizationFilters = useMemo(
    () => ({
      hParams: settings.hParams,
      metric: settings.metric,
      scale: settings.scale,
    }),
    [
      settings.hParams,
      settings.metric,
      settings.scale,
    ],
  );
  
  // Stream available metrics.
  const metrics = useMetricNames(experiments.map((exp) => exp.experiment.id), handleError);
  const selectedScale = settings.scale

  const handleFiltersChange = useCallback(
    (filters: Partial<VisualizationFilters>) => {
      updateSettings(filters);
    },
    [updateSettings],
  );
  
  const handleFiltersReset = useCallback(() => {
    resetSettings();
  }, [resetSettings]);
  
  // Set a default metric of interest filter.
  useEffect(() => {
    if (settings.metric !== undefined) return;
    const activeMetricFound = metrics.find(
      (metric) =>
        metric.type === MetricType.Validation,
    );
    updateSettings({ metric: activeMetricFound ?? metrics.first() });
  }, [experiments, metrics, settings.metric, updateSettings]);
  
  const visualizationFilters = useMemo(() => {
    return (
      <ExperimentVisualizationFilters
        filters={filters}
        fullHParams={settings.hParams}
        metrics={metrics}
        type={ExperimentVisualizationType.HpParallelCoordinates}
        onChange={handleFiltersChange}
        onReset={handleFiltersReset}
      />
    );
  }, [handleFiltersChange, handleFiltersReset, metrics, filters, settings.hParams]);

  const selectedMetric = settings.metric
  const selectedHParams = settings.hParams
  console.log("exps", experiments)
  
  
  const hyperparameters = useMemo(() => {
  const hpMap: Record<string, Hyperparameter> = {}
  experiments.forEach((exp) => {
    const hps = Object.keys(exp.experiment.hyperparameters)
    hps.forEach((hp) => hpMap[hp] = exp.experiment.hyperparameters[hp])
   })
  return hpMap
  }, [experiments])

  console.log("hyperparametsr",hyperparameters)

  const focusedTrial = {id: 1}

  const resetFilteredTrials = useCallback(() => {
    // Skip if there isn't any chart data.
    if (!chartData) return;

    // Initialize a new trial id filter map.
    const newFilteredTrialIdMap = chartData.trialIds.reduce((acc, trialId) => {
      acc[trialId] = true;
      return acc;
    }, {} as Record<number, boolean>);

    // Figure out which trials are filtered out based on user filters.
    Object.entries(hermesCreatedFilters).forEach(([key, list]) => {
      if (!chartData.data[key] || list.length === 0) return;

      chartData.data[key].forEach((value, index) => {
        let isWithinFilter = false;

        list.forEach((filter: Hermes.Filter) => {
          const min = Math.min(Number(filter.value0), Number(filter.value1));
          const max = Math.max(Number(filter.value0), Number(filter.value1));
          if (Number(value) >= min && Number(value) <= max) {
            isWithinFilter = true;
          }
        });

        if (!isWithinFilter) {
          const trialId = chartData.trialIds[index];
          newFilteredTrialIdMap[trialId] = false;
        }
      });
    });

    setFilteredTrialIdMap(newFilteredTrialIdMap);
  }, [chartData, hermesCreatedFilters]);

  useEffect(() => {
    resetFilteredTrials();
  }, [resetFilteredTrials]);

  const config: Hermes.RecursivePartial<Hermes.Config> = useMemo(
    () => ({
      filters: hermesCreatedFilters,
      hooks: {
        onFilterChange: (filters: Hermes.Filters) => {
          // TODO: references are not changing, will need to address this in hermes.
          setHermesCreatedFilters({ ...filters });
        },
        onReset: () => setHermesCreatedFilters({}),
      },
      style: {
        axes: { label: { placement: 'after' } },
        data: {
          series: focusedTrial?.id
            ? chartData?.trialIds.map((trial, index) => ({
                lineWidth: chartData?.trialIds.indexOf(focusedTrial.id) === index ? 3 : 1,
                strokeStyle:
                  chartData?.trialIds.indexOf(focusedTrial.id) === index
                    ? ui.theme.ixOnActive
                    : colorMap[trial]
              }))
            : undefined,
          //targetColorScale: colorScale.map((scale) => scale.color),
          targetDimensionKey: selectedMetric ? metricToStr(selectedMetric) : '',
        },
        dimension: { label: { angle: Math.PI / 4, truncate: 24 } },
        padding: [4, 120, 4, 16],
      },
    }),
    [
      hermesCreatedFilters,
      selectedMetric,
      focusedTrial?.id,
      chartData?.trialIds,
      ui.theme.ixOnActive,
      ui.theme.ixOn,
    ],
  );

  const dimensions = useMemo(() => {
    const newDimensions: Hermes.Dimension[] = selectedHParams.map((key) => {
      const hp = hyperparameters[key] || {};

      if (hp.type === HyperparameterType.Categorical || hp.vals) {
        return {
          categories: hp.vals?.map((val: any) => (isPrimitive(val) ? val : JSON.stringify(val))) ?? [],
          key,
          label: key,
          type: DimensionType.Categorical,
        };
      } else if (hp.type === HyperparameterType.Log) {
        return { key, label: key, logBase: hp.base, type: DimensionType.Logarithmic };
      }

      return { key, label: key, type: DimensionType.Linear };
    });

    // Add metric as column to parcoords dimension list
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

  const clearSelected = useCallback(() => setSelectedRowKeys([]), []);

  useEffect(() => {
    if (!selectedMetric) return;
    const trialMetricsMap: Record<number, number> = {};
    const trialHpTableMap: Record<number, TrialHParams> = {};
    const trialHpMap: Record<string, Record<number, Primitive>> = {};
    console.log("selected metric name")
    console.log(selectedMetric)

        const data: Record<string, Primitive[]> = {};
        let trialMetricRange: Range<number> = defaultNumericRange(true);
       experiments.forEach((exp) => {
          const id = exp.experiment.id;
          const metricValue = exp.bestTrial?.bestValidationMetric?.metrics?.[selectedMetric.name]
          if(!metricValue) return;
          trialMetricsMap[id] =  metricValue;
          trialMetricRange = updateRange<number>(trialMetricRange, metricValue);
          console.log("trials metrics map")
          console.log(exp.bestTrial?.latestValidationMetric?.metrics)
          console.log(trialMetricsMap)
          // This allows for both typical nested hyperparameters and nested categorgical
          // hyperparameter values to be shown, with HpTrialTable deciding which are displayed.
          const flatHParams = { ...exp.bestTrial?.hyperparameters, ...flattenObject(exp.bestTrial?.hyperparameters || {}) };
  
          fullHParams.forEach((hpKey) => {
            if(!flatHParams[hpKey]){
              flatHParams[hpKey] = 0
            }
          })

          Object.keys(flatHParams).forEach((hpKey) => {
            const hpValue = flatHParams[hpKey];
            trialHpMap[hpKey] = trialHpMap[hpKey] ?? {};
            trialHpMap[hpKey][id] = isPrimitive(hpValue) ? hpValue as Primitive : JSON.stringify(hpValue);
            console.log("addimg",hpValue,"to map for id:",id,"and param:",hpKey)
          });

          trialHpTableMap[id] = {
            hparams: clone(flatHParams),
            id,
            metric: metricValue
          };
        });
        
        console.log("triam hp table ")
        console.log(trialHpTableMap)
        console.log("trialHpMap")
        console.log(trialHpMap)

        const trialIds = Object.keys(trialMetricsMap)
          .map((id) => parseInt(id))
          .sort(numericSorter);

        Object.keys(trialHpMap).forEach((hpKey) => {
          data[hpKey] = trialIds.map((trialId) => trialHpMap[hpKey][trialId]);
        });

        // Add metric of interest.
        const metricKey = metricToStr(selectedMetric);
        const metricValues = trialIds.map((id) => trialMetricsMap[id]);
        data[metricKey] = metricValues;

        // Normalize metrics values for parallel coordinates colors.
        const metricRange = getNumericRange(metricValues);

        // Gather hparams for trial table.
        const newTrialHps = trialIds.map((id) => trialHpTableMap[id]);
        setTrialHps(newTrialHps);

        console.log("data")
        console.log(data)
        setChartData({
          data,
          metricRange,
          metricValues,
          trialIds,
        });
  }, [experiments, selectedMetric, fullHParams])
  


  const sendBatchActions = useCallback(
    async (action: Action) => {
      if (action === Action.OpenTensorBoard) {
        return await openOrCreateTensorBoard({
          trialIds: selectedRowKeys,
          workspaceId: workspaceId,
        });
      }
    },
    [selectedRowKeys, experiments],
  );

  const submitBatchAction = useCallback(
    async (action: Action) => {
      try {
        const result = await sendBatchActions(action);
        if (action === Action.OpenTensorBoard && result) {
          openCommandResponse(result as CommandResponse);
        }
      } catch (e) {
        const publicSubject =
          action === Action.OpenTensorBoard
            ? 'Unable to View TensorBoard for Selected Trials'
            : `Unable to ${action} Selected Trials`;
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject,
          silent: false,
          type: ErrorType.Server,
        });
      }
    },
    [sendBatchActions],
  );

  const handleTableRowSelect = useCallback(
    (rowKeys: unknown) => setSelectedRowKeys(rowKeys as number[]),
    [],
  );

  const handleTrialUnselect = useCallback((trialId: number) => {
    setSelectedRowKeys((rowKeys) => rowKeys.filter((id) => id !== trialId));
  }, []);

 //Reset filtered trial ids when HP Viz filters changes.
  useEffect(() => {
    setFilteredTrialIdMap(undefined);
  }, [selectedHParams, selectedMetric]);

  if (pageError) {
    return <Empty description={pageError.message} />;
  } else if (hasLoaded && !chartData) {
    return  (
      <div className={css.waiting}>
        <Alert
          description="Please wait until the experiment is further along."
          message="Not enough data points to plot."
        />
        <Spinner />
      </div>
    );
  }
  console.log("config",config)
  console.log("data", chartData?.data )
  return (
    <div className={css.base}>
      <Section bodyBorder bodyScroll filters={visualizationFilters} >
        <div className={css.container}>
          <div className={css.chart}>
            {experiments.length > 0 && (<ParallelCoordinates
              config={config}
              data={chartData?.data ?? {}}
              dimensions={dimensions}
              disableInteraction={!!focusedTrial}
            />)}
          </div>
          {!focusedTrial && !!selectedMetric && (
            <div>
              <TableBatch
                actions={[
                  { label: Action.OpenTensorBoard, value: Action.OpenTensorBoard },
                  { label: Action.CompareTrials, value: Action.CompareTrials },
                ]}
                selectedRowCount={selectedRowKeys.length}
                onAction={(action) => submitBatchAction(action as Action)}
                onClear={clearSelected}
              />
              {/* <HpTrialTable
                colorScale={colorScale}
                experimentId={1}
                filteredTrialIdMap={filteredTrialIdMap}
                handleTableRowSelect={handleTableRowSelect}
                hyperparameters={hyperparameters}
                metric={selectedMetric}
                selectedRowKeys={selectedRowKeys}
                selection={true}
                trialHps={trialHps}
              /> */}
            </div>
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
