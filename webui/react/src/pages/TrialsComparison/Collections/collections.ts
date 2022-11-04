import { TrialFilters, TrialSorter } from './filters';

export interface TrialsSelection {
  sorter?: TrialSorter;
  trialIds: number[];
}

export interface TrialsCollectionSpec {
  filters: TrialFilters;
  sorter?: TrialSorter;
}

export interface TrialsCollection {
  filters: TrialFilters;
  id: string;
  name: string;
  projectId: string;
  sorter: TrialSorter;
  userId: string;
}

export type TrialsSelectionOrCollection = TrialsSelection | TrialsCollectionSpec;

export const isTrialsSelection = (t: TrialsSelectionOrCollection): t is TrialsSelection =>
  'trialIds' in t;

export const isTrialsCollection = (t: TrialsSelectionOrCollection): t is TrialsCollectionSpec =>
  'filters' in t;

export const getDescriptionText = (t: TrialsSelectionOrCollection): string =>
  isTrialsCollection(t)
    ? 'Filtered Trials'
    : t.trialIds.length === 1
    ? `Trial ${t.trialIds[0]}`
    : `${t.trialIds.length} Trials`;
