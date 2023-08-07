import React, { useMemo, useState } from 'react';

import { ChartGrid, ChartsProps, Serie } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import Spinner from 'components/kit/Spinner';
import { UPlotPoint } from 'components/UPlot/types';
import { closestPointPlugin } from 'components/UPlot/UPlotChart/closestPointPlugin';
import { drawPointsPlugin } from 'components/UPlot/UPlotChart/drawPointsPlugin';
import { tooltipsPlugin } from 'components/UPlot/UPlotChart/tooltipsPlugin';
import { useCheckpointFlow } from 'hooks/useModal/Checkpoint/useCheckpointFlow';
import {
  CheckpointWorkloadExtended,
  ExperimentBase,
  Metric,
  MetricType,
  TrialDetails,
} from 'types';
import handleError from 'utils/error';
import { Loaded, NotLoaded } from 'utils/loadable';
import { metricSorter, metricToKey } from 'utils/metric';

import { useTrialMetrics } from './useTrialMetrics';

export interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

const TRAIN_PREFIX = /^(t_|train_|training_)/;
const VAL_PREFIX = /^(v_|val_|validation_)/;

const isMetricNameMatch = (t: Metric, v: Metric) => {
  const t_stripped = t.name.replace(TRAIN_PREFIX, '');
  const v_stripped = v.name.replace(VAL_PREFIX, '');
  return t_stripped === v_stripped;
};
type XAxisVal = number;
export type CheckpointsDict = Record<XAxisVal, CheckpointWorkloadExtended>;

const TrialDetailsMetrics: React.FC<Props> = ({ experiment, trial }: Props) => {
  const [xAxis, setXAxis] = useState<XAxisDomain>(XAxisDomain.Batches);

  const checkpoint: CheckpointWorkloadExtended | undefined = useMemo(
    () =>
      trial?.bestAvailableCheckpoint
        ? { ...trial.bestAvailableCheckpoint, experimentId: trial?.experimentId, trialId: trial.id }
        : undefined,
    [trial],
  );

  const { contextHolders, openCheckpoint, modelCreateModalComponent, checkpointModalComponent } =
    useCheckpointFlow({
      checkpoint,
      config: experiment.config,
      title: `Best checkpoint for Trial ${trial?.id}`,
    });

  const trials: (TrialDetails | undefined)[] = useMemo(() => [trial], [trial]);

  const {
    metrics,
    isLoaded: isMetricsLoaded,
    data: allData,
    scale,
    setScale,
  } = useTrialMetrics(trials);
  const data = useMemo(() => allData?.[trial?.id || 0], [allData, trial?.id]);

  const checkpointsDict = useMemo<CheckpointsDict>(() => {
    const checkpointXHelpers: Record<XAxisVal, CheckpointWorkloadExtended> = {};
    if (data && checkpoint?.totalBatches) {
      Object.values(data).forEach((metric) => {
        const matchIndex = metric.data[XAxisDomain.Batches]?.findIndex(
          (pt) => pt[0] >= checkpoint.totalBatches,
        );

        if (matchIndex !== undefined && matchIndex >= 0) {
          if (xAxis === XAxisDomain.Time) {
            const timeVals = metric.data[XAxisDomain.Time];
            if (timeVals && timeVals.length > matchIndex) {
              checkpointXHelpers[Math.floor(timeVals[matchIndex][0])] = checkpoint;
            }
          } else if (xAxis === XAxisDomain.Batches) {
            const batchX = metric.data[XAxisDomain.Batches]?.[matchIndex][0];
            if (batchX) {
              checkpointXHelpers[batchX] = checkpoint;
            }
          }
        }
      });
    }
    return checkpoint?.totalBatches ? checkpointXHelpers : {};
  }, [data, checkpoint, xAxis]);

  const pairedMetrics: ([Metric] | [Metric, Metric])[] | undefined = useMemo(() => {
    const val = metrics.filter((m) => m.type === MetricType.Validation).sort(metricSorter);
    const train = metrics.filter((m) => m.type === MetricType.Training).sort(metricSorter);
    let out: ([Metric] | [Metric, Metric])[] = [];
    while (val.length) {
      const v = val.shift();
      if (!v) return;
      let pair: [Metric] | [Metric, Metric] = [v];
      const t_match = train.findIndex((t) => isMetricNameMatch(t, v));
      if (t_match !== -1) pair = train.splice(t_match, 1).concat(pair) as [Metric, Metric];
      out.push(pair);
    }
    out = out.concat(train.map((t) => [t]));
    return out;
  }, [metrics]);

  const chartsProps = useMemo(() => {
    if (!isMetricsLoaded) {
      return NotLoaded;
    }

    const out: ChartsProps = [];

    pairedMetrics?.forEach(([trainingMetric, valMetric]) => {
      // this code doesnt depend on their being training or validation metrics
      // naming just makes it easier to read
      const trainingMetricKey = metricToKey(trainingMetric);
      const trainingMetricSeries = data?.[trainingMetricKey];
      if (!trainingMetricSeries) return;

      const series: Serie[] = [trainingMetricSeries];

      if (valMetric) {
        const valMetricKey = metricToKey(valMetric);
        const valMetricSeries = data?.[valMetricKey];
        if (valMetricSeries) series.push(valMetricSeries);
      }

      const xValSet = new Set<number>();
      series.forEach((serie) => {
        serie.data[xAxis]?.forEach((pt) => {
          xValSet.add(pt[0]);
        });
      });
      const xVals = Array.from(xValSet).sort((a, b) => a - b);

      const onPointClick = (event: MouseEvent, point: UPlotPoint) => {
        const xVal = xVals[point.idx];
        const selectedCheckpoint =
          xVal !== undefined ? checkpointsDict[Math.floor(xVal)] : undefined;
        if (selectedCheckpoint) {
          openCheckpoint();
        }
      };

      out.push({
        onPointClick,
        plugins: [
          closestPointPlugin({
            checkpointsDict,
            onPointClick,
            yScale: 'y',
          }),
          drawPointsPlugin(checkpointsDict),
          tooltipsPlugin({
            getXTooltipHeader(xIndex) {
              const xVal = xVals[xIndex];
              if (xVal === undefined) return '';
              const checkpoint = checkpointsDict?.[Math.floor(xVal)];
              if (!checkpoint) return '';
              return '<div>⬦ Best Checkpoint <em>(click to view details)</em> </div>';
            },
            isShownEmptyVal: false,
            seriesColors: series.map((s) => s.color ?? '#009BDE'),
          }),
        ],
        series,
        title: trainingMetric.name.replace(TRAIN_PREFIX, '').replace(VAL_PREFIX, ''),
        xAxis,
        xLabel: String(xAxis),
      });
    });
    return Loaded(out);
  }, [pairedMetrics, isMetricsLoaded, data, xAxis, checkpointsDict, openCheckpoint]);

  return (
    <>
      {isMetricsLoaded ? (
        <ChartGrid
          chartsProps={chartsProps}
          handleError={handleError}
          scale={scale}
          setScale={setScale}
          xAxis={xAxis}
          onXAxisChange={setXAxis}
        />
      ) : (
        <Spinner spinning />
      )}
      {contextHolders.map((contextHolder, i) => (
        <React.Fragment key={i}>{contextHolder}</React.Fragment>
      ))}
      {modelCreateModalComponent}
      {checkpointModalComponent}
    </>
  );
};

export default TrialDetailsMetrics;
