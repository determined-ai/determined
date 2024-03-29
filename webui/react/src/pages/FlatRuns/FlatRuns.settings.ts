import * as t from 'io-ts';

import { INIT_FORMSET } from 'components/FilterForm/components/FilterFormStore';
import { SettingsConfig } from 'hooks/useSettings';
import { DEFAULT_SELECTION, SelectionType } from 'pages/F_ExpList/F_ExperimentList.settings';

import { defaultColumnWidths, defaultRunColumns } from './columns';

// have to intersect with an empty object bc of settings store type issue
export const FlatRunsSettings = t.intersection([
  t.type({}),
  t.type({
    columns: t.array(t.string),
    columnWidths: t.record(t.string, t.number),
    compare: t.boolean,
    filterset: t.string, // save FilterFormSet as string
    pageLimit: t.number,
    pinnedColumnsCount: t.number,
    selection: SelectionType,
    sortString: t.string,
  }),
]);
export type FlatRunsSettings = t.TypeOf<typeof FlatRunsSettings>;

export const ProjectUrlSettings = t.partial({
  compare: t.boolean,
  page: t.number,
});

export const settingsConfigForProject = (id: number): SettingsConfig<FlatRunsSettings> => ({
  settings: {
    columns: {
      defaultValue: defaultRunColumns,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: t.array(t.string),
    },
    columnWidths: {
      defaultValue: defaultColumnWidths,
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: t.record(t.string, t.number),
    },
    compare: {
      defaultValue: false,
      storageKey: 'compare',
      type: t.boolean,
    },
    filterset: {
      defaultValue: JSON.stringify(INIT_FORMSET),
      skipUrlEncoding: true,
      storageKey: 'filterset',
      type: t.string,
    },
    pageLimit: {
      defaultValue: 20,
      skipUrlEncoding: true,
      storageKey: 'pageLimit',
      type: t.number,
    },
    pinnedColumnsCount: {
      defaultValue: 3,
      skipUrlEncoding: true,
      storageKey: 'pinnedColumnsCount',
      type: t.number,
    },
    selection: {
      defaultValue: DEFAULT_SELECTION,
      skipUrlEncoding: true,
      storageKey: 'selection',
      type: SelectionType,
    },
    sortString: {
      defaultValue: 'id=desc',
      skipUrlEncoding: true,
      storageKey: 'sortString',
      type: t.string,
    },
  },
  storagePath: `flatRunsForProject${id}`,
});
