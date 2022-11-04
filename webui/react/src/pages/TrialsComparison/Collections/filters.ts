export interface NumberRange {
  max?: string;
  min?: string;
}

export type NumberRangeDict = Record<string, NumberRange>;

export interface TrialSorter {
  sortDesc: boolean;
  // `${namespace}.${field}`
  sortKey: string;
}

export interface Ranker {
  rank?: string;
  sorter: TrialSorter;
}

export interface TrialFilters {
  experimentIds?: string[];
  hparams?: NumberRangeDict;
  projectIds?: string[];
  ranker?: Ranker;
  searcher?: string;
  states?: string[];
  tags?: string[];
  trainingMetrics?: NumberRangeDict;
  trialIds?: string[];
  userIds?: string[];
  validationMetrics?: NumberRangeDict;
  workspaceIds?: string[];
}

export type FilterSetter = (prev: TrialFilters) => TrialFilters;

export type SetFilters = (fs: FilterSetter) => void;
