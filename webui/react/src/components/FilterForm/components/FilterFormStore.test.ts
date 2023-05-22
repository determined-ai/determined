import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import {
  Conjunction,
  FilterFormSet,
  FilterFormSetWithoutId,
  FormField,
  FormGroup,
  FormKind,
  Operator,
} from 'components/FilterForm/components/type';
import { V1ColumnType, V1LocationType, V1ProjectColumn } from 'services/api-ts-sdk';

// Remove `id` property from object
const jsonReplacer = (key: string, value: unknown): unknown => {
  return key === 'id' ? undefined : value;
};

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
    it('should initialize store without init data', () => {
      const filterFormStore = new FilterFormStore();
      const jsonWithId = filterFormStore.formset.get();
      const asJsonString = filterFormStore.asJsonString.get();

      expect(jsonWithId).toStrictEqual({
        filterGroup: {
          children: [],
          conjunction: Conjunction.And,
          id: ROOT_ID,
          kind: FormKind.Group,
        },
        showArchived: false,
      });
      expect(asJsonString).toStrictEqual(JSON.stringify(EMPTY_DATA));

      expect(filterFormStore.fieldCount.get()).toBe(0);
    });

    it('should initialize store with init data', () => {
      const filterFormStore = new FilterFormStore();
      filterFormStore.init(initData);

      expect(filterFormStore.formset.get()).toStrictEqual(initData);
      expect(filterFormStore.fieldCount.get()).toBe(6);
    });

    it('should deep clone init data to avoid unexpected data overwrite', () => {
      const filterFormStore = new FilterFormStore();
      filterFormStore.init(initData);
      filterFormStore.addChild(ROOT_ID, FormKind.Field);
      const jsonWithId = filterFormStore.formset.get();

      expect(jsonWithId).not.toStrictEqual(initData);
    });
  });

  describe('Data Interaction', () => {
    describe('Basic Field and Group Interaction', () => {
      it('should sweep invalid groups and conditions', () => {
        const filterFormStore = new FilterFormStore();
        filterFormStore.init(initData);
        filterFormStore.sweep();
        expect(filterFormStore.fieldCount.get()).toBe(5);
      });

      it('should add new fields', () => {
        const filterFormStore = new FilterFormStore();
        filterFormStore.addChild(ROOT_ID, FormKind.Field);

        expect(
          JSON.parse(JSON.stringify(filterFormStore.formset.get(), jsonReplacer)),
        ).toStrictEqual({
          filterGroup: {
            children: [
              {
                columnName: 'name',
                kind: FormKind.Field,
                location: V1LocationType.EXPERIMENT,
                operator: Operator.Contains,
                type: V1ColumnType.TEXT,
                value: null,
              },
            ],
            conjunction: Conjunction.And,
            kind: FormKind.Group,
          },
          showArchived: false,
        });

        filterFormStore.addChild(ROOT_ID, FormKind.Field);

        expect(
          JSON.parse(JSON.stringify(filterFormStore.formset.get(), jsonReplacer)),
        ).toStrictEqual({
          filterGroup: {
            children: [
              {
                columnName: 'name',
                kind: FormKind.Field,
                location: V1LocationType.EXPERIMENT,
                operator: Operator.Contains,
                type: V1ColumnType.TEXT,
                value: null,
              },
              {
                columnName: 'name',
                kind: FormKind.Field,
                location: V1LocationType.EXPERIMENT,
                operator: Operator.Contains,
                type: V1ColumnType.TEXT,
                value: null,
              },
            ],
            conjunction: Conjunction.And,
            kind: FormKind.Group,
          },
          showArchived: false,
        });

        expect(filterFormStore.asJsonString.get()).toStrictEqual(JSON.stringify(EMPTY_DATA));
      });

      it('should add new groups', () => {
        const filterFormStore = new FilterFormStore();
        filterFormStore.addChild(ROOT_ID, FormKind.Group);

        expect(
          JSON.parse(JSON.stringify(filterFormStore.formset.get(), jsonReplacer)),
        ).toStrictEqual({
          filterGroup: {
            children: [{ children: [], conjunction: Conjunction.And, kind: FormKind.Group }],
            conjunction: Conjunction.And,
            kind: FormKind.Group,
          },
          showArchived: false,
        });

        filterFormStore.addChild(ROOT_ID, FormKind.Group);

        expect(
          JSON.parse(JSON.stringify(filterFormStore.formset.get(), jsonReplacer)),
        ).toStrictEqual({
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

        const jsonWithId = filterFormStore.formset.get();
        const group = jsonWithId.filterGroup.children[1];
        if (group.kind === FormKind.Group) {
          filterFormStore.addChild(group.id, FormKind.Group);

          expect(
            JSON.parse(JSON.stringify(filterFormStore.formset.get(), jsonReplacer)),
          ).toStrictEqual({
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
        const filterFormStore = new FilterFormStore();
        filterFormStore.addChild(ROOT_ID, FormKind.Group);
        filterFormStore.addChild(ROOT_ID, FormKind.Field);

        expect(
          JSON.parse(JSON.stringify(filterFormStore.formset.get(), jsonReplacer)),
        ).toStrictEqual({
          filterGroup: {
            children: [
              { children: [], conjunction: Conjunction.And, kind: FormKind.Group },
              {
                columnName: 'name',
                kind: FormKind.Field,
                location: V1LocationType.EXPERIMENT,
                operator: Operator.Contains,
                type: V1ColumnType.TEXT,
                value: null,
              },
            ],
            conjunction: Conjunction.And,
            kind: FormKind.Group,
          },
          showArchived: false,
        });

        expect(filterFormStore.asJsonString.get()).toStrictEqual(JSON.stringify(EMPTY_DATA));

        const jsonWithId = filterFormStore.formset.get();
        const group = jsonWithId.filterGroup.children[1];
        if (group.kind === FormKind.Group) {
          filterFormStore.addChild(group.id, FormKind.Field);
          filterFormStore.addChild(group.id, FormKind.Group);

          expect(
            JSON.parse(JSON.stringify(filterFormStore.formset.get(), jsonReplacer)),
          ).toStrictEqual({
            filterGroup: {
              children: [
                {
                  columnName: 'name',
                  kind: FormKind.Field,
                  location: V1LocationType.EXPERIMENT,
                  operator: Operator.Contains,
                  type: V1ColumnType.TEXT,
                  value: null,
                },
                {
                  children: [
                    { children: [], conjunction: Conjunction.And, kind: FormKind.Group },
                    {
                      columnName: 'name',
                      kind: FormKind.Field,
                      location: V1LocationType.EXPERIMENT,
                      operator: Operator.Contains,
                      type: V1ColumnType.TEXT,
                      value: null,
                    },
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
        const filterFormStore = new FilterFormStore();
        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        const json = filterFormStore.formset.get();
        expect(json.filterGroup.children.length).toBe(4);
        const thirdFieldId = json.filterGroup.children[2].id;
        filterFormStore.removeChild(thirdFieldId);
        expect(filterFormStore.formset.get().filterGroup.children).not.toContain(thirdFieldId);
      });

      it('should remove empty group', () => {
        const filterFormStore = new FilterFormStore();
        filterFormStore.addChild(ROOT_ID, FormKind.Group);
        const groupId = filterFormStore.formset.get().filterGroup.children[0].id;
        filterFormStore.removeChild(groupId);
        expect(
          JSON.parse(JSON.stringify(filterFormStore.formset.get(), jsonReplacer)),
        ).toStrictEqual(EMPTY_DATA);
        expect(filterFormStore.asJsonString.get()).toStrictEqual(JSON.stringify(EMPTY_DATA));
      });

      it('should remove non-empty group', () => {
        const filterFormStore = new FilterFormStore();
        filterFormStore.addChild(ROOT_ID, FormKind.Group);
        const group = filterFormStore.formset.get().filterGroup.children[0];
        if (group.kind === FormKind.Group) {
          filterFormStore.addChild(group.id, FormKind.Field);
          filterFormStore.removeChild(group.id);
          expect(filterFormStore.asJsonString.get()).toStrictEqual(JSON.stringify(EMPTY_DATA));
        }
      });

      it('should clear all (remove ROOT_ID)', () => {
        const filterFormStore = new FilterFormStore();
        filterFormStore.removeChild(ROOT_ID);
        expect(filterFormStore.asJsonString.get()).toStrictEqual(JSON.stringify(EMPTY_DATA));
        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        filterFormStore.removeChild(ROOT_ID);
        expect(filterFormStore.asJsonString.get()).toStrictEqual(JSON.stringify(EMPTY_DATA));

        const filterFormStoreWithInit = new FilterFormStore();
        filterFormStoreWithInit.init(initData);
        filterFormStoreWithInit.removeChild(ROOT_ID);
        expect(filterFormStoreWithInit.asJsonString.get()).toStrictEqual(
          JSON.stringify(EMPTY_DATA),
        );
      });

      it('should change `show archived` value', () => {
        const filterFormStore = new FilterFormStore();
        filterFormStore.setArchivedValue(true);
        expect(filterFormStore.formset.get().showArchived).toBeTruthy();
        filterFormStore.setArchivedValue(false);
        expect(filterFormStore.formset.get().showArchived).toBeFalsy();
        filterFormStore.setArchivedValue(false);
        expect(filterFormStore.formset.get().showArchived).toBeFalsy();
      });

      it('should `show archived` value remain the same after clear all', () => {
        const filterFormStoreWithInit = new FilterFormStore();
        filterFormStoreWithInit.init(initData);
        expect(filterFormStoreWithInit.formset.get().showArchived).toBeFalsy();
        filterFormStoreWithInit.setArchivedValue(true);
        expect(filterFormStoreWithInit.formset.get().showArchived).toBeTruthy();
        filterFormStoreWithInit.removeChild(ROOT_ID);
        expect(filterFormStoreWithInit.formset.get().showArchived).toBeTruthy();
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
        const filterFormStore = new FilterFormStore();

        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        filterFormStore.addChild(ROOT_ID, FormKind.Field);
        filterFormStore.addChild(ROOT_ID, FormKind.Field, { index: 2, item: item });
        const fields = filterFormStore.formset.get().filterGroup.children;
        expect(fields).toHaveLength(3);
        expect(fields[2].id).toBe(ID);
        // move index2 to index0
        filterFormStore.removeChild(fields[2].id);
        expect(filterFormStore.formset.get().filterGroup.children).toHaveLength(2);
        filterFormStore.addChild(ROOT_ID, FormKind.Field, { index: 0, item: item });
        expect(filterFormStore.formset.get().filterGroup.children).toHaveLength(3);
        expect(filterFormStore.formset.get().filterGroup.children[0].id).toBe(ID);

        // to make sure the original object is not referenced (should not be shallow copy)
        filterFormStore.setFieldValue(ID, 'value');
        expect(item.value).toBeNull();
      });

      it('should move field into different group', () => {
        const filterFormStore = new FilterFormStore();

        filterFormStore.addChild(ROOT_ID, FormKind.Group);
        filterFormStore.addChild(ROOT_ID, FormKind.Field, { index: 2, item: item });
        const fields = filterFormStore.formset.get().filterGroup.children;
        expect(fields).toHaveLength(2);
        expect(fields[1].id).toBe(ID);
        // move index2 to index0
        filterFormStore.removeChild(ID);
        expect(filterFormStore.formset.get().filterGroup.children).toHaveLength(1);
        filterFormStore.addChild(fields[0].id, FormKind.Field, { index: 0, item: item });
        const group = filterFormStore.formset.get().filterGroup.children[0];
        if (group.kind === FormKind.Group) {
          expect(filterFormStore.formset.get().filterGroup.children).toHaveLength(1);
          expect(group.children).toHaveLength(1);
          expect(group.children[0].id).toBe(ID);

          // to make sure the original object is not referenced (should not be shallow copy)
          filterFormStore.setFieldValue(ID, 'value');
          expect(item.value).toBeNull();
        }
      });
    });

    describe('Field Value Interaction', () => {
      it('should change column name', () => {
        const filterFormStore = new FilterFormStore();
        filterFormStore.addChild(ROOT_ID, FormKind.Field);

        const field: Readonly<FormGroup | FormField> =
          filterFormStore.formset.get().filterGroup.children[0];
        if (field.kind === FormKind.Field) {
          expect(field.columnName).toBe('name');
        }
        const col: V1ProjectColumn = {
          column: 'id',
          displayName: 'ID',
          location: V1LocationType.EXPERIMENT,
          type: 'COLUMN_TYPE_NUMBER',
        };
        filterFormStore.setFieldColumnName(field.id, col);
        if (field.kind === FormKind.Field) {
          expect(field.columnName).toBe('id');
        }
      });

      it('should change operator', () => {
        const filterFormStore = new FilterFormStore();
        filterFormStore.addChild(ROOT_ID, FormKind.Field);

        const field: Readonly<FormGroup | FormField> =
          filterFormStore.formset.get().filterGroup.children[0];
        if (field.kind === FormKind.Field) {
          expect(field.operator).toBe(Operator.Contains);
          filterFormStore.setFieldOperator(field.id, Operator.GreaterEq);
          expect(field.operator).toBe(Operator.GreaterEq);
        }
      });

      it('should change value', () => {
        const filterFormStore = new FilterFormStore();
        filterFormStore.addChild(ROOT_ID, FormKind.Field);

        const field: Readonly<FormGroup | FormField> =
          filterFormStore.formset.get().filterGroup.children[0];
        if (field.kind === FormKind.Field) {
          const value = 'test';
          expect(field.value).toBeNull();
          filterFormStore.setFieldValue(field.id, value);
          expect(field.value).toBe(value);
        }
      });
    });

    describe('Group Value Interaction', () => {
      it('should change conjugation', () => {
        const filterFormStore = new FilterFormStore();
        filterFormStore.addChild(ROOT_ID, FormKind.Group);

        const field: Readonly<FormGroup | FormField> =
          filterFormStore.formset.get().filterGroup.children[0];
        if (field.kind === FormKind.Group) {
          expect(field.conjunction).toBe(Conjunction.And);
          filterFormStore.setFieldConjunction(field.id, Conjunction.Or);
          expect(field.conjunction).toBe(Conjunction.Or);
        }
      });
    });
  });
});
