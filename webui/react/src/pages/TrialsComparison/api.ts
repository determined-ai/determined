import {
  NumberRange,
  NumberRangeDict,
  TrialFilters,
  TrialSorter,
} from 'pages/TrialsComparison/Collections/filters';
import {
  TrialSorterNamespace,
  Trialv1State,
  V1ColumnFilter,
  V1OrderBy,
  V1TrialFilters,
  V1TrialsCollection,
  V1TrialSorter,
  V1TrialTag,
} from 'services/api-ts-sdk';
import { finiteElseUndefined, isFiniteNumber } from 'utils/data';
import { camelCaseToSnake, snakeCaseToCamelCase } from 'utils/string';

import { TrialsCollection } from './Collections/collections';

export const encodeTrialSorter = (s?: TrialSorter): V1TrialSorter => {
  if (!s?.sortKey)
    return {
      field: 'searcher_metric_loss',
      namespace: TrialSorterNamespace.UNSPECIFIED,
      orderBy: V1OrderBy.ASC,
    };

  const prefix = s.sortKey.split('.')[0];
  const namespace =
    prefix === 'hparams'
      ? TrialSorterNamespace.HPARAMS
      : prefix === 'validationMetrics'
      ? TrialSorterNamespace.VALIDATIONMETRICS
      : prefix === 'trainingMetrics'
      ? TrialSorterNamespace.TRAININGMETRICS
      : TrialSorterNamespace.UNSPECIFIED;

  const rawField = s.sortKey === 'searcherMetricValue' ? 'searcherMetricLoss' : s.sortKey;

  const field =
    namespace === TrialSorterNamespace.UNSPECIFIED
      ? camelCaseToSnake(rawField)
      : rawField.split('.').slice(1).join('.');

  return {
    field,
    namespace: namespace,
    orderBy: s.sortDesc ? V1OrderBy.DESC : V1OrderBy.ASC,
  };
};

const prefixForNamespace: Record<TrialSorterNamespace, string> = {
  [TrialSorterNamespace.HPARAMS]: 'hparams',
  [TrialSorterNamespace.UNSPECIFIED]: '',
  [TrialSorterNamespace.TRAININGMETRICS]: 'training_metrics',
  [TrialSorterNamespace.VALIDATIONMETRICS]: 'validation_metrics',
};

export const decodeTrialSorter = (s?: V1TrialSorter): TrialSorter => {
  if (!s)
    return {
      sortDesc: false,
      sortKey: 'searcherMetricValue',
    };

  const prefix = snakeCaseToCamelCase(prefixForNamespace[s.namespace]);

  const rawField = s.field === 'searcherMetricLoss' ? 'searcherMetricValue' : s.field;

  return {
    sortDesc: s.orderBy === V1OrderBy.DESC,
    sortKey: prefix ? [prefix, rawField].join('.') : snakeCaseToCamelCase(rawField),
  };
};

export const encodeIdList = (l?: string[]): number[] | undefined =>
  l?.map((i) => parseInt(i)).filter(isFiniteNumber);

const encodeNumberRangeDict = (d: NumberRangeDict): Array<V1ColumnFilter> =>
  Object.entries(d).map(([key, range]) => ({
    filter: {
      gte: finiteElseUndefined((range as NumberRange).min),
      lte: finiteElseUndefined((range as NumberRange).max),
    },
    name: key,
  }));

const decodeNumberRangeDict = (d: Array<V1ColumnFilter>): NumberRangeDict =>
  d
    .map((f) =>
      f.name
        ? {
            [f.name]: {
              max: f.filter?.lte ? String(f.filter?.lte) : undefined,
              min: f.filter?.gte ? String(f.filter?.gte) : undefined,
            },
          }
        : {},
    )
    .reduce((a, b) => ({ ...a, ...b }), {});

export const encodeFilters = (f: TrialFilters): V1TrialFilters => {
  return {
    experimentIds: encodeIdList(f.experimentIds),
    hparams: encodeNumberRangeDict(f.hparams ?? {}),
    projectIds: encodeIdList(f.projectIds),
    rankWithinExp: f.ranker?.rank
      ? {
          rank: finiteElseUndefined(f.ranker.rank),
          sorter: encodeTrialSorter(f.ranker.sorter),
        }
      : undefined,
    searcher: f.searcher,
    states: f.states as unknown as Trialv1State[],
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
  states: f.states ? (f.states as unknown as string[]) : undefined,
  tags: f.tags?.map((tag: V1TrialTag) => tag.key),
  trainingMetrics: decodeNumberRangeDict(f.trainingMetrics ?? []),
  trialIds: f.trialIds?.map(String),
  userIds: f.userIds?.map(String),
  validationMetrics: decodeNumberRangeDict(f.validationMetrics ?? []),
  workspaceIds: f.workspaceIds?.map(String),
});

export const decodeTrialsCollection = (c: V1TrialsCollection): TrialsCollection => ({
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
