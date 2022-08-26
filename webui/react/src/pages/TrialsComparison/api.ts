import {
  NumberRange,
  NumberRangeDict,
  TrialFilters,
  TrialSorter,
} from 'pages/TrialsComparison/Collections/filters';
import {
  Determinedtrialv1State,
  TrialSorterNamespace,
  V1NumberRangeFilter,
  V1OrderBy,
  V1TrialFilters,
  V1TrialsCollection,
  V1TrialSorter,
  V1TrialTag,
} from 'services/api-ts-sdk';
import {
  isNumber,
  numberElseUndefined,
} from 'shared/utils/data';
import { camelCaseToSnake, snakeCaseToCamelCase } from 'shared/utils/string';

import { TrialsCollection } from './Collections/collections';

export const encodeTrialSorter = (s?: TrialSorter): V1TrialSorter => {
  if (!s?.sortKey) return {
    field: 'trial_id',
    namespace: TrialSorterNamespace.TRIALS,
    orderBy: V1OrderBy.DESC,
  };

  const prefix = s.sortKey.split('.')[0];
  const namespace = (
    prefix === 'hparams'
      ? TrialSorterNamespace.HPARAMS
      : prefix === 'validationMetrics'
        ? TrialSorterNamespace.VALIDATIONMETRICS
        : prefix === 'trainingMetrics'
          ? TrialSorterNamespace.TRAININGMETRICS
          : TrialSorterNamespace.TRIALS);

  const field = namespace === TrialSorterNamespace.TRIALS
    ? camelCaseToSnake(s.sortKey)
    : s.sortKey.split('.').slice(1).join('.');

  return {
    field,
    namespace: namespace,
    orderBy: s.sortDesc ? V1OrderBy.DESC : V1OrderBy.ASC,
  };
};

const prefixForNamespace: Record<TrialSorterNamespace, string> = {
  [TrialSorterNamespace.HPARAMS]: 'hparams',
  [TrialSorterNamespace.TRIALS]: '',
  [TrialSorterNamespace.TRAININGMETRICS]: 'training_metrics',
  [TrialSorterNamespace.VALIDATIONMETRICS]: 'validation_metrics',
};

export const decodeTrialSorter = (s?: V1TrialSorter): TrialSorter => {
  if (!s) return {
    sortDesc: true,
    sortKey: 'trialId',
  };

  const prefix = prefixForNamespace[s.namespace];

  return {
    sortDesc: s.orderBy === V1OrderBy.DESC,
    sortKey: prefix ? [ prefix, s.field ].join('.') : snakeCaseToCamelCase(s.field),
  };

};

export const encodeIdList = (l?: string[]): number[] | undefined =>
  l?.map((i) => parseInt(i)).filter((i) => isNumber(i));

const encodeNumberRangeDict = (d: NumberRangeDict): Array<V1NumberRangeFilter> =>
  Object.entries(d).map(([ key, range ]) => ({
    max: numberElseUndefined((range as NumberRange).max),
    min: numberElseUndefined((range as NumberRange).min),
    name: key,
  }));

const decodeNumberRangeDict = (d: Array<V1NumberRangeFilter>): NumberRangeDict =>
  d.map((f) => (
    f.name ? {
      [f.name]: {
        max: f.max ? String(f.max) : undefined,
        min: f.min ? String(f.min) : undefined,
      },
    } : {})).reduce((a, b) => ({ ...a, ...b }), {});

export const encodeFilters = (f: TrialFilters): V1TrialFilters => {
  return {
    experimentIds: encodeIdList(f.experimentIds),
    hparams: encodeNumberRangeDict(f.hparams ?? {}),
    projectIds: encodeIdList(f.projectIds),
    rankWithinExp: f.ranker?.rank
      ? {
        rank: numberElseUndefined(f.ranker.rank),
        sorter: encodeTrialSorter(f.ranker.sorter),
      }
      : undefined,
    searcher: f.searcher,
    states: f.states as unknown as Determinedtrialv1State[],
    tags: f.tags?.map((tag: string) => ({ key: tag })),
    trainingMetrics: encodeNumberRangeDict(f.trainingMetrics ?? {}),
    trialIds: encodeIdList(f.trialIds),
    userIds: encodeIdList(f.userIds),
    validationMetrics: encodeNumberRangeDict(f.validationMetrics ?? {}),
    workspaceIds: encodeIdList(f.workspaceIds),
  };
};
export const decodeFilters = (f: V1TrialFilters): TrialFilters => ({
  experimentIds: f.experimentIds?.map(String),
  hparams: decodeNumberRangeDict(f.hparams ?? []),
  projectIds: f.projectIds?.map(String),
  ranker: {
    rank: String(f.rankWithinExp?.rank ?? 0),
    sorter: decodeTrialSorter(f.rankWithinExp?.sorter),
  },
  searcher: f.searcher,
  states: f.states ? f.states as unknown as string[] : undefined,
  tags: f.tags?.map((tag: V1TrialTag) => tag.key),
  trainingMetrics: decodeNumberRangeDict(f.trainingMetrics ?? []),
  trialIds: f.trialIds?.map(String),
  userIds: f.userIds?.map(String),
  validationMetrics: decodeNumberRangeDict(f.validationMetrics ?? []),
  workspaceIds: f.workspaceIds?.map(String),
});

export const decodeTrialsCollection = (c: V1TrialsCollection): TrialsCollection =>
  ({
    filters: decodeFilters(c.filters),
    id: String(c.id),
    name: c.name,
    projectId: String(c.projectId),
    sorter: decodeTrialSorter(c.sorter),
    userId: String(c.userId),
  });

export const encodeTrialsCollection = (c: TrialsCollection): V1TrialsCollection => ({
  filters: encodeFilters(c.filters),
  id: parseInt(c.id),
  name: c.name,
  projectId: parseInt(c.projectId),
  sorter: encodeTrialSorter(c.sorter),
  userId: parseInt(c.userId),
});
