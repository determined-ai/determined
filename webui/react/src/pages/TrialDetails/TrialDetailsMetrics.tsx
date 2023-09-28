import React, { useMemo, useState } from 'react';

import { ChartGrid, ChartsProps } from 'components/kit/LineChart';
import Spinner from 'components/kit/Spinner';
import { Loaded, NotLoaded } from 'components/kit/utils/loadable';
import { UPlotPoint } from 'components/UPlot/types';
import { closestPointPlugin } from 'components/UPlot/UPlotChart/closestPointPlugin';
import { drawPointsPlugin } from 'components/UPlot/UPlotChart/drawPointsPlugin';
import { tooltipsPlugin } from 'components/UPlot/UPlotChart/tooltipsPlugin';
import { useCheckpointFlow } from 'hooks/useModal/Checkpoint/useCheckpointFlow';
import {
  CheckpointWorkloadExtended,
  ExperimentBase,
  Metric,
  Serie,
  TrialDetails,
  XAxisDomain,
} from 'types';
import handleError from 'utils/error';
import { metricSorter, metricToKey } from 'utils/metric';

import { useTrialMetrics } from './useTrialMetrics';

export interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

type XAxisVal = number;
export type CheckpointsDict = Record<XAxisVal, CheckpointWorkloadExtended>;

const TRAIN_PREFIX = /^(t_|train_|training_)/;
const VAL_PREFIX = /^(v_|val_|validation_)/;

const stripPrefix = (metricName: string): string => {
  return metricName.replace(TRAIN_PREFIX, '').replace(VAL_PREFIX, '');
};

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

  const groupedMetrics: Metric[][] = useMemo(() => {
    const map = metrics.reduce((acc, metric) => {
      const metricName = stripPrefix(metric.name);
      acc[metricName] = acc[metricName] ?? [];
      acc[metricName].push(metric);
      return acc;
    }, {} as Record<string, Metric[]>);
    return Object.keys(map)
      .sort()
      .map((metricName) => map[metricName].sortAll(metricSorter));
  }, [metrics]);

  const chartsProps = useMemo(() => {
    if (!isMetricsLoaded) return NotLoaded;

    const out: ChartsProps = [];

    groupedMetrics.forEach((groupMetrics) => {
      const series: Serie[] = groupMetrics
        .map((metric) => data?.[metricToKey(metric)])
        .filter((metricData) => !!metricData);

      const xValSet = series.reduce((set, serie) => {
        serie.data[xAxis]?.forEach((point) => set.add(point[0]));
        return set;
      }, new Set<number>());
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
        title: groupMetrics.length !== 0 ? stripPrefix(groupMetrics[0].name) : 'No Metric Name',
        xAxis,
        xLabel: String(xAxis),
      });
    });
    return Loaded(out);
  }, [groupedMetrics, isMetricsLoaded, data, xAxis, checkpointsDict, openCheckpoint]);

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
