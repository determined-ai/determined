import * as t from 'io-ts';

import { INIT_FORMSET } from 'components/FilterForm/components/FilterFormStore';
import { DEFAULT_SELECTION, SelectionType } from 'pages/F_ExpList/F_ExperimentList.settings';

import { defaultColumnWidths, defaultRunColumns } from './columns';

// have to intersect with an empty object bc of settings store type issue
export const FlatRunsSettings = t.type({
  columns: t.array(t.string),
  columnWidths: t.record(t.string, t.number),
  compare: t.boolean,
  filterset: t.string, // save FilterFormSet as string
  pageLimit: t.number,
  pinnedColumnsCount: t.number,
  selection: SelectionType,
  sortString: t.string,
});
export type FlatRunsSettings = t.TypeOf<typeof FlatRunsSettings>;

export const defaultFlatRunsSettings: Required<FlatRunsSettings> = {
  columns: defaultRunColumns,
  columnWidths: defaultColumnWidths,
  compare: false,
  filterset: JSON.stringify(INIT_FORMSET),
  pageLimit: 20,
  pinnedColumnsCount: 3,
  selection: DEFAULT_SELECTION,
  sortString: 'id=desc',
};

export const ProjectUrlSettings = t.partial({
  compare: t.boolean,
  page: t.number,
});

export const settingsPathForProject = (id: number): string => `flatRunsForProject${id}`;
