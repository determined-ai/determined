import { Loadable, NotLoaded } from 'hew/utils/loadable';

import { FilterFormStore, getInitField } from 'components/FilterForm/components/FilterFormStore';
import {
  Conjunction,
  FilterFormSet,
  FilterFormSetWithoutId,
  FormField,
  FormKind,
  Operator,
} from 'components/FilterForm/components/type';
import { V1ColumnType, V1LocationType, V1ProjectColumn } from 'services/api-ts-sdk';

// Remove `id` property from object
const jsonReplacer = (key: string, value: unknown): unknown => {
  return key === 'id' ? undefined : value;
};

const initField = JSON.parse(JSON.stringify(getInitField(V1LocationType.EXPERIMENT), jsonReplacer));

const ROOT_ID = 'ROOT';

const EMPTY_DATA: Readonly<FilterFormSetWithoutId> = {
  filterGroup: { children: [], conjunction: Conjunction.And, kind: FormKind.Group },
  showArchived: false,
};

const initData: Readonly<FilterFormSet> = {
  filterGroup: {
    children: [
      {
        columnName: 'name',
        id: '7857e7c4-4eef-4e8b-82ae-6eba5ca200bd',
        kind: FormKind.Field,
        location: V1LocationType.EXPERIMENT,
        operator: Operator.Contains,
        type: V1ColumnType.TEXT,
        value: 'test',
      },
      {
        columnName: 'name',
        id: '0c474949-0985-40ef-a287-8b99c7f09d02',
        kind: FormKind.Field,
        location: V1LocationType.EXPERIMENT,
        operator: Operator.Contains,
        type: V1ColumnType.TEXT,
        value: 'name',
      },
      {
        columnName: 'forkedFrom',
        id: 'b4677888-7bf9-4068-acab-1d48abe3ee30',
        kind: FormKind.Field,
        location: V1LocationType.EXPERIMENT,
        operator: Operator.NotEq,
        type: V1ColumnType.NUMBER,
        value: 123,
      },
      {
        children: [
          {
            columnName: 'name',
            id: '82ca6e46-fa34-4815-81f1-530127580371',
            kind: FormKind.Field,
            location: V1LocationType.EXPERIMENT,
            operator: Operator.Contains,
            type: V1ColumnType.TEXT,
            value: 'name',
          },
        ],
        conjunction: Conjunction.And,
        id: '112b20a7-6221-4ba9-9d00-9ee9f0649058',
        kind: FormKind.Group,
      },
      {
        children: [
          {
            children: [
              {
                columnName: 'name',
                id: '69d9c920-511f-4580-b2ae-1a9fe7844d57',
                kind: FormKind.Field,
                location: V1LocationType.EXPERIMENT,
                operator: Operator.Contains,
                type: V1ColumnType.TEXT,
                value: 'name',
              },
            ],
            conjunction: Conjunction.And,
            id: 'db1811b0-455a-4121-9e76-2075d40f1169',
            kind: FormKind.Group,
          },
          {
            columnName: 'name',
            id: 'e49cc15c-27c8-475c-9446-2e613f982193',
            kind: FormKind.Field,
            location: V1LocationType.EXPERIMENT,
            operator: Operator.Contains,
            type: V1ColumnType.TEXT,
            value: null,
          },
        ],
        conjunction: Conjunction.And,
        id: '7f7335d1-e084-4492-ad04-9e7e80e0cc69',
        kind: FormKind.Group,
      },
    ],
    conjunction: Conjunction.And,
    id: ROOT_ID,
    kind: FormKind.Group,
  },
  showArchived: false,
};

describe('FilterFormStore', () => {
  describe('Init', () => {
    it('should initialize store as NotLoaded', () => {
      const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
      expect(filterFormStore.formset.get()).toEqual(NotLoaded);
    });

    it('should have an empty init() fill with default values', () => {
      const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
      filterFormStore.init();
      const loadableFormset = filterFormStore.formset.get();

      const jsonWithId = Loadable.getOrElse(null, loadableFormset);
      expect(jsonWithId).toStrictEqual({
        filterGroup: {
          children: [],
          conjunction: Conjunction.And,
          id: ROOT_ID,
          kind: FormKind.Group,
        },
        showArchived: false,
      });

      const asJsonString = filterFormStore.asJsonString.get();
      expect(asJsonString).toStrictEqual(JSON.stringify(EMPTY_DATA));

      expect(filterFormStore.fieldCount.get()).toBe(0);
    });

    it('should initialize store with init data', () => {
      const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
      filterFormStore.init(initData);

      const loadableFormset = filterFormStore.formset.get();
      const formset = Loadable.getOrElse(null, loadableFormset);
      expect(formset).toStrictEqual(initData);
      expect(filterFormStore.fieldCount.get()).toBe(5);
    });

    it('should deep clone init data to avoid unexpected data overwrite', () => {
      const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
      filterFormStore.init(initData);
      filterFormStore.addChild(ROOT_ID, FormKind.Field);
      const jsonWithId = filterFormStore.formset.get();

      expect(jsonWithId).not.toStrictEqual(initData);
    });
  });

  describe('Data Interaction', () => {
    const getFormset = (filterFormStore: FilterFormStore) => {
      const loadableFormset = filterFormStore.formset.get();
      if (!Loadable.isLoaded(loadableFormset)) {
        throw new Error('formstore is unloaded');
      }
      return loadableFormset.data;
    };
    const getField = (filterFormStore: FilterFormStore) => {
      return getFormset(filterFormStore).filterGroup.children[0];
    };

    describe('Basic Field and Group Interaction', () => {
      it('should sweep invalid groups and conditions', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init(initData);
        filterFormStore.sweep();
        expect(filterFormStore.fieldCount.get()).toBe(5);
      });

      it('should add new fields', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init();
        filterFormStore.addChild(ROOT_ID, FormKind.Field);

        let formset = getFormset(filterFormStore);
        expect(JSON.parse(JSON.stringify(formset, jsonReplacer))).toStrictEqual({
          filterGroup: {
            children: [initField],
            conjunction: Conjunction.And,
            kind: FormKind.Group,
          },
          showArchived: false,
        });

        filterFormStore.addChild(ROOT_ID, FormKind.Field);

        formset = getFormset(filterFormStore);
        expect(JSON.parse(JSON.stringify(formset, jsonReplacer))).toStrictEqual({
          filterGroup: {
            children: [initField, initField],
            conjunction: Conjunction.And,
            kind: FormKind.Group,
          },
          showArchived: false,
        });

        expect(filterFormStore.asJsonString.get()).toStrictEqual(JSON.stringify(EMPTY_DATA));
      });

      it('should add new groups', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init();
        filterFormStore.addChild(ROOT_ID, FormKind.Group);

        let formset = getFormset(filterFormStore);
        expect(JSON.parse(JSON.stringify(formset, jsonReplacer))).toStrictEqual({
          filterGroup: {
            children: [{ children: [], conjunction: Conjunction.And, kind: FormKind.Group }],
            conjunction: Conjunction.And,
            kind: FormKind.Group,
          },
          showArchived: false,
        });

        filterFormStore.addChild(ROOT_ID, FormKind.Group);

        formset = getFormset(filterFormStore);
        expect(JSON.parse(JSON.stringify(formset, jsonReplacer))).toStrictEqual({
          filterGroup: {
            children: [
              { children: [], conjunction: Conjunction.And, kind: FormKind.Group },
              { children: [], conjunction: Conjunction.And, kind: FormKind.Group },
            ],
            conjunction: Conjunction.And,
            kind: FormKind.Group,
          },
          showArchived: false,
        });

        const loadableJsonWithId = filterFormStore.formset.get();
        const jsonWithId = Loadable.getOrElse(null, loadableJsonWithId);
        const group = jsonWithId?.filterGroup?.children?.[1];
        if (group?.kind === FormKind.Group) {
          filterFormStore.addChild(group.id, FormKind.Group);

          formset = getFormset(filterFormStore);
          expect(JSON.parse(JSON.stringify(formset, jsonReplacer))).toStrictEqual({
            filterGroup: {
              children: [
                { children: [], conjunction: Conjunction.And, kind: FormKind.Group },
                {
                  children: [{ children: [], conjunction: Conjunction.And, kind: FormKind.Group }],
                  conjunction: Conjunction.And,
                  kind: FormKind.Group,
                },
              ],
              conjunction: Conjunction.And,
              kind: FormKind.Group,
            },
            showArchived: false,
          });
        }
      });

      it('should add new fields/group comprehensively', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init();
        filterFormStore.addChild(ROOT_ID, FormKind.Group);
        filterFormStore.addChild(ROOT_ID, FormKind.Field);

        const loadableFormset = filterFormStore.formset.get();
        let formset = Loadable.getOrElse(null, loadableFormset);
        expect(JSON.parse(JSON.stringify(formset, jsonReplacer))).toStrictEqual({
          filterGroup: {
            children: [
              { children: [], conjunction: Conjunction.And, kind: FormKind.Group },
              initField,
            ],
            conjunction: Conjunction.And,
            kind: FormKind.Group,
          },
          showArchived: false,
        });

        expect(filterFormStore.asJsonString.get()).toStrictEqual(JSON.stringify(EMPTY_DATA));

        const loadableJsonWithId = filterFormStore.formset.get();
        const jsonWithId = Loadable.getOrElse(null, loadableJsonWithId);
        const group = jsonWithId?.filterGroup?.children?.[1];
        if (group?.kind === FormKind.Group) {
          filterFormStore.addChild(group.id, FormKind.Field);
          filterFormStore.addChild(group.id, FormKind.Group);

          formset = Loadable.getOrElse(null, loadableFormset);
          expect(JSON.parse(JSON.stringify(formset, jsonReplacer))).toStrictEqual({
            filterGroup: {
              children: [
                initField,
                {
                  children: [
                    { children: [], conjunction: Conjunction.And, kind: FormKind.Group },
                    initField,
                  ],
                  conjunction: Conjunction.And,
                  kind: FormKind.Group,
                },
              ],
              conjunction: Conjunction.And,
              kind: FormKind.Group,
            },
            showArchived: false,
          });

          expect(filterFormStore.asJsonString.get()).toStrictEqual(JSON.stringify(EMPTY_DATA));
        }
      });

      it('should remove field', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init();
        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        const loadableJson = filterFormStore.formset.get();
        const json = Loadable.getOrElse(null, loadableJson);
        expect(json?.filterGroup?.children?.length).toBe(4);
        const thirdFieldId = json?.filterGroup?.children?.[2]?.id;
        if (thirdFieldId) {
          filterFormStore.removeChild(thirdFieldId);
        }
        const loadableFormset = filterFormStore.formset.get();
        const formSet = Loadable.getOrElse(null, loadableFormset);
        expect(formSet?.filterGroup?.children).not.toContain(thirdFieldId);
      });

      it('should remove empty group', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init();
        filterFormStore.addChild(ROOT_ID, FormKind.Group);
        let formset = getFormset(filterFormStore);
        const groupId = formset?.filterGroup?.children?.[0]?.id;
        if (groupId) {
          filterFormStore.removeChild(groupId);
        }

        formset = getFormset(filterFormStore);
        expect(JSON.parse(JSON.stringify(formset, jsonReplacer))).toStrictEqual(EMPTY_DATA);
        expect(filterFormStore.asJsonString.get()).toStrictEqual(JSON.stringify(EMPTY_DATA));
      });

      it('should remove non-empty group', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init();
        filterFormStore.addChild(ROOT_ID, FormKind.Group);
        const loadableFormset = filterFormStore.formset.get();
        const formSet = Loadable.getOrElse(null, loadableFormset);
        const group = formSet?.filterGroup?.children?.[0];
        if (group?.kind === FormKind.Group) {
          filterFormStore.addChild(group.id, FormKind.Field);
          filterFormStore.removeChild(group.id);
          expect(filterFormStore.asJsonString.get()).toStrictEqual(JSON.stringify(EMPTY_DATA));
        }
      });

      it('should clear all (remove ROOT_ID)', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init();
        filterFormStore.removeChild(ROOT_ID);
        expect(filterFormStore.asJsonString.get()).toStrictEqual(JSON.stringify(EMPTY_DATA));
        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        filterFormStore.removeChild(ROOT_ID);
        expect(filterFormStore.asJsonString.get()).toStrictEqual(JSON.stringify(EMPTY_DATA));

        const filterFormStoreWithInit = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStoreWithInit.init(initData);
        filterFormStoreWithInit.removeChild(ROOT_ID);
        expect(filterFormStoreWithInit.asJsonString.get()).toStrictEqual(
          JSON.stringify(EMPTY_DATA),
        );
      });

      it('should change `show archived` value', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init();
        filterFormStore.setArchivedValue(true);
        let loadableFormset = filterFormStore.formset.get();
        let formset = Loadable.getOrElse(null, loadableFormset);
        expect(formset?.showArchived).toBeTruthy();
        filterFormStore.setArchivedValue(false);
        loadableFormset = filterFormStore.formset.get();
        formset = Loadable.getOrElse(null, loadableFormset);
        expect(formset?.showArchived).toBeFalsy();
        filterFormStore.setArchivedValue(false);
        loadableFormset = filterFormStore.formset.get();
        formset = Loadable.getOrElse(null, loadableFormset);
        expect(formset?.showArchived).toBeFalsy();
      });

      it('should `show archived` value remain the same after clear all', () => {
        const filterFormStoreWithInit = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStoreWithInit.init(initData);
        let loadableFormset = filterFormStoreWithInit.formset.get();
        let formset = Loadable.getOrElse(null, loadableFormset);
        expect(formset?.showArchived).toBeFalsy();
        filterFormStoreWithInit.setArchivedValue(true);
        loadableFormset = filterFormStoreWithInit.formset.get();
        formset = Loadable.getOrElse(null, loadableFormset);
        expect(formset?.showArchived).toBeTruthy();
        filterFormStoreWithInit.removeChild(ROOT_ID);
        loadableFormset = filterFormStoreWithInit.formset.get();
        formset = Loadable.getOrElse(null, loadableFormset);
        expect(formset?.showArchived).toBeTruthy();
      });
    });

    describe('Order Field and Group Interaction', () => {
      const ID = 'unique_id';
      const item: FormField = {
        columnName: 'name',
        id: ID,
        kind: FormKind.Field,
        location: V1LocationType.EXPERIMENT,
        operator: Operator.Contains,
        type: V1ColumnType.TEXT,
        value: null,
      };

      it('should change the field order', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init();

        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        filterFormStore.addChild(ROOT_ID, FormKind.Field, { index: 2, item: item });
        let formset = getFormset(filterFormStore);
        const fields = formset.filterGroup.children;
        expect(fields).toHaveLength(3);
        expect(fields[2].id).toBe(ID);
        // move index2 to index0
        filterFormStore.removeChild(fields[2].id);
        formset = getFormset(filterFormStore);
        expect(formset.filterGroup.children).toHaveLength(2);
        filterFormStore.addChild(ROOT_ID, FormKind.Field, { index: 0, item: item });
        formset = getFormset(filterFormStore);
        expect(formset.filterGroup.children).toHaveLength(3);
        expect(formset.filterGroup.children[0].id).toBe(ID);

        // to make sure the original object is not referenced (should not be shallow copy)
        filterFormStore.setFieldValue(ID, 'value');
        expect(item.value).toBeNull();
      });

      it('should move field into different group', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init();

        filterFormStore.addChild(ROOT_ID, FormKind.Group);
        filterFormStore.addChild(ROOT_ID, FormKind.Field, { index: 2, item: item });
        let formset = getFormset(filterFormStore);
        const fields = formset.filterGroup.children;
        expect(formset.filterGroup.children).toHaveLength(2);
        expect(fields[1]?.id).toBe(ID);

        // move index2 to index0
        filterFormStore.removeChild(ID);
        formset = getFormset(filterFormStore);
        expect(formset?.filterGroup?.children).toHaveLength(1);

        filterFormStore.addChild(fields?.[0]?.id ?? '', FormKind.Field, { index: 0, item: item });
        formset = getFormset(filterFormStore);
        const group = formset?.filterGroup?.children?.[0];
        if (group && group.kind === FormKind.Group) {
          expect(formset?.filterGroup?.children).toHaveLength(1);
          expect(group.children).toHaveLength(1);
          expect(group.children[0].id).toBe(ID);

          // to make sure the original object is not referenced (should not be shallow copy)
          formset = getFormset(filterFormStore);
          filterFormStore.setFieldValue(ID, 'value');
          expect(item.value).toBeNull();
        }
      });
    });

    describe('Field Value Interaction', () => {
      it('should change column name', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init();
        filterFormStore.addChild(ROOT_ID, FormKind.Field);

        let field = getField(filterFormStore);
        expect(field).toMatchObject({ columnName: initField.columnName });
        const col: V1ProjectColumn = {
          column: 'id',
          displayName: 'ID',
          location: V1LocationType.EXPERIMENT,
          type: 'COLUMN_TYPE_NUMBER',
        };
        filterFormStore.setFieldColumnName(field.id, col);
        field = getField(filterFormStore);
        expect(field).toMatchObject({ columnName: 'id' });
      });

      it('should change operator', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init();
        filterFormStore.addChild(ROOT_ID, FormKind.Field);

        let field = getField(filterFormStore);
        expect(field).toMatchObject({ operator: Operator.Eq });
        filterFormStore.setFieldOperator(field.id, Operator.GreaterEq);
        field = getField(filterFormStore);
        expect(field).toMatchObject({ operator: Operator.GreaterEq });
      });

      it('should change value', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init();
        filterFormStore.addChild(ROOT_ID, FormKind.Field);

        let field = getField(filterFormStore);
        const value = 'test';
        expect(field).toMatchObject({ value: null });
        filterFormStore.setFieldValue(field.id, value);
        field = getField(filterFormStore);
        expect(field).toMatchObject({ value });
      });
    });

    describe('Group Value Interaction', () => {
      it('should change conjunction', () => {
        const filterFormStore = new FilterFormStore(V1LocationType.EXPERIMENT);
        filterFormStore.init();
        filterFormStore.addChild(ROOT_ID, FormKind.Group);

        let field = getField(filterFormStore);
        expect(field).toMatchObject({ conjunction: Conjunction.And });
        filterFormStore.setFieldConjunction(field.id, Conjunction.Or);
        field = getField(filterFormStore);
        expect(field).toMatchObject({ conjunction: Conjunction.Or });
      });
    });
  });
});
