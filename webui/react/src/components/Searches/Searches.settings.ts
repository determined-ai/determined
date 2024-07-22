import * as t from 'io-ts';

import { INIT_FORMSET } from 'components/FilterForm/components/FilterFormStore';
import { RegularSelectionType, SelectionType } from 'types';

import { defaultColumnWidths, defaultExperimentColumns } from './columns';

export const DEFAULT_SELECTION: t.TypeOf<typeof RegularSelectionType> = {
  selections: [],
  type: 'ONLY_IN',
};

// have to intersect with an empty object bc of settings store type issue
export const ProjectSettings = t.intersection([
  t.type({}),
  t.partial({
    columns: t.array(t.string),
    columnWidths: t.record(t.string, t.number),
    compare: t.boolean,
    filterset: t.string, // save FilterFormSet as string
    heatmapOn: t.boolean,
    heatmapSkipped: t.array(t.string),
    pageLimit: t.number,
    pinnedColumnsCount: t.number,
    selection: SelectionType,
    sortString: t.string,
  }),
]);
export type ProjectSettings = t.TypeOf<typeof ProjectSettings>;

export const ProjectUrlSettings = t.partial({
  compare: t.boolean,
  page: t.number,
});

export const settingsPathForProject = (id: number): string => `searchesForProject${id}`;
export const defaultProjectSettings: Required<ProjectSettings> = {
  columns: defaultExperimentColumns,
  columnWidths: defaultColumnWidths,
  compare: false,
  filterset: JSON.stringify(INIT_FORMSET),
  heatmapOn: false,
  heatmapSkipped: [],
  pageLimit: 20,
  pinnedColumnsCount: 3,
  selection: DEFAULT_SELECTION,
  sortString: 'id=desc',
};
