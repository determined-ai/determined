import { Observable, observable } from 'micro-observables';
import { v4 as uuidv4 } from 'uuid';

import { V1ColumnType, V1LocationType, V1ProjectColumn } from 'services/api-ts-sdk';

import {
  AvaliableOperators,
  Conjunction,
  FilterFormSet,
  FilterFormSetWithoutId,
  FormField,
  FormFieldValue,
  FormGroup,
  FormKind,
  Operator,
} from './type';

export const ITEM_LIMIT = 50;

const INIT_FORMSET: Readonly<FilterFormSet> = {
  filterGroup: { children: [], conjunction: Conjunction.And, id: 'ROOT', kind: FormKind.Group },
  showArchived: false,
};

const getInitGroup = (): FormGroup => ({
  children: [],
  conjunction: Conjunction.And,
  id: uuidv4(),
  kind: FormKind.Group,
});

const getInitField = (): FormField => ({
  columnName: 'name',
  id: uuidv4(),
  kind: FormKind.Field,
  location: V1LocationType.EXPERIMENT,
  operator: AvaliableOperators[V1ColumnType.TEXT][0],
  type: V1ColumnType.TEXT,
  value: null,
});

export class FilterFormStore {
  #formset = observable<FilterFormSet>(structuredClone(INIT_FORMSET));

  constructor(data?: Readonly<FilterFormSet>) {
    if (data) {
      this.#formset = observable<FilterFormSet>(structuredClone(data));
    }
  }

  public get formset(): Observable<Readonly<FilterFormSet>> {
    return this.#formset.readOnly();
  }

  public get json(): Readonly<FilterFormSet> {
    const filterFormSet: Readonly<FilterFormSet> = this.#formset.get();
    return filterFormSet;
  }

  public get jsonWithoutId(): Readonly<FilterFormSetWithoutId> {
    const replacer = (key: string, value: unknown): unknown => {
      return key === 'id' ? undefined : value;
    };
    const filterFormSet = this.#formset.get();
    return JSON.parse(JSON.stringify(filterFormSet, replacer));
  }

  #getFormById(filterGroup: FormGroup, id: string): FormField | FormGroup | undefined {
    const recur = (form: FormGroup | FormField): FormGroup | FormField | undefined => {
      if (form.id === id) {
        return form;
      }
      if (form.kind === FormKind.Group && form.children.length === 0) {
        return undefined;
      }

      if (form.kind === FormKind.Group) {
        for (const child of form.children) {
          const ans = recur(child);
          if (ans) {
            return ans;
          }
        }
      }
      return undefined;
    };

    return recur(filterGroup);
  }

  public setFieldColumnName(
    id: string,
    col: Pick<V1ProjectColumn, 'location' | 'type' | 'column'>,
  ): void {
    const filterSet: Readonly<FilterFormSet> = this.#formset.get();
    const filterGroup = filterSet.filterGroup;
    const ans = this.#getFormById(filterGroup, id);
    if (ans && ans.kind === FormKind.Field) {
      ans.columnName = col.column;
      ans.location = col.location;
      ans.type = col.type;
      this.#formset.set({ filterGroup, showArchived: filterSet.showArchived });
    }
  }

  public setFieldOperator(id: string, operator: Operator): void {
    const filterSet: Readonly<FilterFormSet> = this.#formset.get();
    const filterGroup = filterSet.filterGroup;
    const ans = this.#getFormById(filterGroup, id);
    if (ans && ans.kind === FormKind.Field && Object.values(Operator).includes(operator)) {
      ans.operator = operator;
      this.#formset.set({ filterGroup, showArchived: filterSet.showArchived });
    }
  }

  public setFieldConjunction(id: string, conjunction: Conjunction): void {
    const filterSet: Readonly<FilterFormSet> = this.#formset.get();
    const filterGroup = filterSet.filterGroup;
    const ans = this.#getFormById(filterGroup, id);
    if (ans && ans.kind === FormKind.Group && Object.values(Conjunction).includes(conjunction)) {
      ans.conjunction = conjunction;
      this.#formset.set({ filterGroup, showArchived: filterSet.showArchived });
    }
  }

  public setFieldValue(id: string, value: FormFieldValue): void {
    const filterSet: Readonly<FilterFormSet> = this.#formset.get();
    const filterGroup = filterSet.filterGroup;
    const ans = this.#getFormById(filterGroup, id);
    if (ans && ans.kind === FormKind.Field) {
      ans.value = value;
      this.#formset.set({ filterGroup, showArchived: filterSet.showArchived });
    }
  }

  public addChild(
    id: string,
    addType: FormKind,
    obj?: { index: number; item: Readonly<FormGroup | FormField> },
  ): void {
    const filterSet: Readonly<FilterFormSet> = this.#formset.get();
    const filterGroup = filterSet.filterGroup;
    const recur = (form: FormGroup | FormField): void => {
      if (form.id === id && form.kind === FormKind.Group) {
        if (obj) {
          form.children.splice(obj.index, 0, obj.item);
        } else {
          form.children.push(addType === FormKind.Group ? getInitGroup() : getInitField());
        }
        return;
      }

      if (form.kind === FormKind.Group) {
        for (const child of form.children) {
          recur(child);
        }
      }
    };

    recur(filterGroup);
    this.#formset.set({ filterGroup, showArchived: filterSet.showArchived });
  }

  public removeChild(id: string): void {
    const filterSet: Readonly<FilterFormSet> = this.#formset.get();
    const filterGroup = filterSet.filterGroup;

    if (filterGroup.id === id) {
      // if remove top group
      this.#formset.set({ ...structuredClone(INIT_FORMSET), showArchived: filterSet.showArchived });
      return;
    }

    const recur = (form: FormGroup | FormField): void => {
      if (form.kind === FormKind.Group) {
        const prevLength = form.children.length;
        form.children = form.children.filter((c) => c.id !== id);
        if (prevLength === form.children.length) {
          for (const child of form.children) {
            recur(child);
          }
        }
      }
    };
    recur(filterGroup);
    this.#formset.set({ filterGroup, showArchived: filterSet.showArchived });
  }

  public setArchivedValue(val: boolean): void {
    const filterSet: Readonly<FilterFormSet> = this.#formset.get();
    this.#formset.set({ ...filterSet, showArchived: val });
  }
}
