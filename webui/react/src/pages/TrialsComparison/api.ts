import {
  NumberRange,
  NumberRangeDict,
  TrialFilters,
  TrialSorter,
} from 'pages/TrialsComparison/Collections/filters';
import {
  TrialSorterNamespace,
  V1NumberRangeFilter,
  V1OrderBy,
  V1TrialFilters,
  V1TrialsCollection,
  V1TrialSorter,
  V1TrialTag,
  Determinedtrialv1State
} from 'services/api-ts-sdk';
import {
  isNumber,
  numberElseUndefined,
} from 'shared/utils/data';
import { camelCaseToSnake } from 'shared/utils/string';

import { TrialsCollection } from './Collections/collections';

export const encodeTrialSorter = (s?: TrialSorter): V1TrialSorter => {
  if (!s?.sortKey) return {
    field: 'trial_id',
    namespace: TrialSorterNamespace.TRIALSUNSPECIFIED,
    orderBy: V1OrderBy.DESC,
  };

  const prefix = s.sortKey.split('.')[0];
  const namespace = (
    prefix === 'hparams'
      ? TrialSorterNamespace.TRIALSUNSPECIFIED
      : prefix === 'validation_metrics'
        ? TrialSorterNamespace.VALIDATIONMETRICS
        : prefix === 'training_metrics'
          ? TrialSorterNamespace.TRAININGMETRICS
          : TrialSorterNamespace.TRIALSUNSPECIFIED);
  const field = namespace === TrialSorterNamespace.TRIALSUNSPECIFIED
    ? s.sortKey
    : s.sortKey.split('.').slice(1).join('.');

  return {
    field: camelCaseToSnake(field),
    namespace: namespace,
    orderBy: s.orderBy,
  };
};

const prefixForNamespace: Record<TrialSorterNamespace, string> = {
  [TrialSorterNamespace.HPARAMS]: 'hparams',
  [TrialSorterNamespace.TRIALSUNSPECIFIED]: '',
  [TrialSorterNamespace.TRAININGMETRICS]: 'training_metrics',
  [TrialSorterNamespace.VALIDATIONMETRICS]: 'validation_metrics',
};

export const decodeTrialSorter = (s?: V1TrialSorter): TrialSorter => {
  if (!s) return {
    orderBy: V1OrderBy.DESC,
    sortKey: 'trialId',
  };

  const prefix = prefixForNamespace[s.namespace];

  return {
    orderBy: s.orderBy ?? V1OrderBy.DESC,
    sortKey: prefix ? [ prefix, s.field ].join('.') : s.field,
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

  const encodeStates = (states: string[]): Determinedtrialv1State[] => {
    const apiStateToTrialStateMap: Record< string, Determinedtrialv1State> = {
    'ACTIVE' : Determinedtrialv1State.ACTIVEUNSPECIFIED,
    'PAUSED':Determinedtrialv1State.PAUSED,
    'KILLED':Determinedtrialv1State.STOPPINGKILLED,
    'COMPLETE': Determinedtrialv1State.STOPPINGCOMPLETED,
    'CANCELED': Determinedtrialv1State.CANCELED,
    'COMPLETED': Determinedtrialv1State.COMPLETED,
    'ERROR':Determinedtrialv1State.ERROR
    }
    return states.map((s) => apiStateToTrialStateMap[s])
  } 
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
    tags: f.tags?.map((tag: string) => ({ key: tag, value: '1' })),
    trainingMetrics: encodeNumberRangeDict(f.trainingMetrics ?? {}),
    userIds: encodeIdList(f.userIds),
    validationMetrics: encodeNumberRangeDict(f.validationMetrics ?? {}),
    workspaceIds: encodeIdList(f.workspaceIds),
    states: encodeStates(f.state ?? [])  
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
  tags: f.tags?.map((tag: V1TrialTag) => tag.key),
  trainingMetrics: decodeNumberRangeDict(f.trainingMetrics ?? []),
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
