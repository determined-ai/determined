import { V1AugmentedTrial, V1QueryTrialsResponse } from 'services/api-ts-sdk';
import { Primitive, RawJson } from 'shared/types';
import { clone, flattenObject } from 'shared/utils/data';
import { union } from 'shared/utils/set';
import {
  Metric,
  MetricType,
} from 'types';
import { metricKeyToMetric, metricToKey } from 'utils/metric';

export function mergeLists<T>(
  A: Array<T>,
  B: Array<T>,
  equalFn: (a: T, b: T) => boolean = (a: T, b: T) => a === b,
): Array<T> {
  return [ ...A, ...B.filter((b) => A.every((a) => !equalFn(a, b))) ];

}

// `${type}|${name}`
type MetricKey = string

const valMapForHParams = (hparams: RawJson): HpValsMap =>
  Object.entries(flattenObject(hparams || {}))
    .map(([ key, value ]) => ({ [String(key)]: new Set([ value ]) }))
    .reduce((a, b) => ({ ...a, ...b }), {});

const mergeHpValMaps = (A: HpValsMap, B: HpValsMap): HpValsMap => {
  const hps = mergeLists(Object.keys(A), Object.keys(B));
  return hps.map((hp) => ({ [hp]: union(A[hp] ?? new Set(), B[hp] ?? new Set()) }))
    .reduce((a, b) => ({ ...a, ...b }), {});
};

const aggregateHpVals = (agg: HpValsMap, hparams: RawJson) =>
  mergeHpValMaps(agg, valMapForHParams(hparams));

const decodeMetricKeys = (metricsData: RawJson, type: MetricType): Record<MetricKey, boolean> =>
  Object.keys(metricsData)
    .map((name) => ({ name, type }))
    .map((m) => ({ [metricToKey(m)]: true }))
    .reduce((a, b) => ({ ...a, ...b }), {});

export type HpValsMap = Record<string, Set<Primitive>>

export interface TrialsWithMetadata {
  data: V1AugmentedTrial[];
  hparams: HpValsMap;
  ids: number[];
  maxBatch: number;
  metricKeys: Record<MetricKey, boolean>;
  metrics: Metric[];
}

export const aggregrateTrialsMetadata =
(agg: TrialsWithMetadata, trial: V1AugmentedTrial): TrialsWithMetadata => {
  const tMetrics = decodeMetricKeys(trial.trainingMetrics, MetricType.Training);
  const vMetrics = decodeMetricKeys(trial.validationMetrics, MetricType.Validation);

  return {
    data: [ ...agg.data, { ...trial, hparams: flattenObject(trial.hparams) } ],
    hparams: aggregateHpVals(agg.hparams, trial.hparams),
    ids: [ ...agg.ids, trial.trialId ],
    maxBatch: Math.max(agg.maxBatch, trial.totalBatches),
    metricKeys: { ...agg.metricKeys, ...tMetrics, ...vMetrics },
    metrics: [],
  };
};

export const defaultTrialData = {
  data: [],
  hparams: {},
  ids: [],
  maxBatch: 1,
  metricKeys: {},
  metrics: [],
  total: 0,
};

export const decodeTrialsWithMetadata = (
  response?: V1QueryTrialsResponse,
): TrialsWithMetadata => {
  if (!response?.trials) return clone(defaultTrialData);
  const t = response.trials?.reduce(aggregrateTrialsMetadata, clone(defaultTrialData));

  // const tmpFunc = (k) => {
  //   const foo = metricKeyToMetric(k);
  //   return {
  //     ...foo,
  //     log: [],
  //     set type(type: string) {
  //       this.log.push(type);
  //     },
  //   };
  // };

  const metrics = Object.keys(t.metricKeys)
    .map(metricKeyToMetric) as Metric[];

  return { ...t, metrics };
};
