import { observable, Observable, WritableObservable } from 'micro-observables';
import { v4 as uuidv4 } from 'uuid';

import {
  AvailableOperators,
  Conjunction,
  FilterFormSet,
  FilterFormSetWithoutId,
  FormField,
  FormFieldValue,
  FormGroup,
  FormKind,
  Operator,
} from 'components/FilterForm/components/type';
import { V1ColumnType, V1LocationType, V1ProjectColumn } from 'services/api-ts-sdk';

export const ITEM_LIMIT = 50;

export const ROOT_ID = 'ROOT';

export const INIT_FORMSET: Readonly<FilterFormSet> = {
  filterGroup: { children: [], conjunction: Conjunction.And, id: ROOT_ID, kind: FormKind.Group },
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
  operator: AvailableOperators[V1ColumnType.TEXT][0],
  type: V1ColumnType.TEXT,
  value: null,
});

export class FilterFormStore {
  #formset: WritableObservable<FilterFormSet> = observable(structuredClone(INIT_FORMSET));

  public init(data?: Readonly<FilterFormSet>): void {
    this.#formset.update(() => structuredClone(data ? data : INIT_FORMSET));
  }

  public get formset(): Observable<Readonly<FilterFormSet>> {
    return this.#formset.readOnly();
  }

  public get asJsonString(): Observable<string> {
    const replacer = (key: string, value: unknown): unknown => {
      return key === 'id' ? undefined : value;
    };
    return this.#formset.select((formset) => {
      const sweepedForm = this.#sweepInvalid(structuredClone(formset.filterGroup));
      const newFormSet: FilterFormSetWithoutId = JSON.parse(
        JSON.stringify({ ...formset, filterGroup: sweepedForm }, replacer),
      );
      return JSON.stringify(newFormSet);
    });
  }

  public get fieldCount(): Observable<number> {
    const countFields = (form: Readonly<FormGroup>): number => {
      let count = 0;
      for (const child of form.children) {
        count += child.kind === FormKind.Group ? countFields(child) : 1;
      }
      return count;
    };
    return this.#formset.select((formset) => countFields(formset.filterGroup));
  }

  #isValid(form: Readonly<FormGroup | FormField>): boolean {
    if (form.kind === FormKind.Field) {
      return (
        form.operator === Operator.IsEmpty ||
        form.operator === Operator.NotEmpty ||
        form.value != null
      );
    } else {
      return form.children.length > 0;
    }
  }

  // remove invalid groups and conditions
  #sweepInvalid = (form: FormGroup): Readonly<FormGroup> => {
    const sweepRecur = (form: FormGroup): Readonly<FormGroup> => {
      const children = form.children.filter(this.#isValid); // remove unused groups and conditions
      for (let child of children) {
        if (child.kind === FormKind.Group) {
          child = sweepRecur(child); // recursively remove groups and conditions
        }
      }
      form.children = children.filter(this.#isValid); // double check for groups
      return form;
    };

    // clone form to avoid reference change
    return sweepRecur(structuredClone(form));
  };

  // remove invalid groups and conditions and then store sweeped data in #formset
  public sweep(): void {
    this.#formset.update((prev) => {
      const filterGroup = this.#sweepInvalid(prev.filterGroup);
      return { ...prev, filterGroup };
    });
  }

  #getFormById(filterGroup: FormGroup, id: string): FormField | FormGroup | undefined {
    const traverse = (form: FormGroup | FormField): FormGroup | FormField | undefined => {
      if (form.id === id) {
        return form;
      }
      if (form.kind === FormKind.Group && form.children.length === 0) {
        return undefined;
      }

      if (form.kind === FormKind.Group) {
        for (const child of form.children) {
          const ans = traverse(child);
          if (ans) {
            return ans;
          }
        }
      }
      return undefined;
    };

    return traverse(filterGroup);
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
      this.#formset.update((prev) => ({ ...prev, filterGroup }));
    }
  }

  public setFieldOperator(id: string, operator: Operator): void {
    const filterSet: Readonly<FilterFormSet> = this.#formset.get();
    const filterGroup = filterSet.filterGroup;
    const ans = this.#getFormById(filterGroup, id);
    if (ans && ans.kind === FormKind.Field && Object.values(Operator).includes(operator)) {
      ans.operator = operator;
      this.#formset.update((prev) => ({ ...prev, filterGroup }));
    }
  }

  public setFieldConjunction(id: string, conjunction: Conjunction): void {
    const filterSet: Readonly<FilterFormSet> = this.#formset.get();
    const filterGroup = filterSet.filterGroup;
    const ans = this.#getFormById(filterGroup, id);
    if (ans && ans.kind === FormKind.Group && Object.values(Conjunction).includes(conjunction)) {
      ans.conjunction = conjunction;
      this.#formset.update((prev) => ({ ...prev, filterGroup }));
    }
  }

  public setFieldValue(id: string, value: FormFieldValue): void {
    const filterSet: Readonly<FilterFormSet> = this.#formset.get();
    const filterGroup = filterSet.filterGroup;
    const ans = this.#getFormById(filterGroup, id);
    if (ans && ans.kind === FormKind.Field) {
      ans.value = value;
      this.#formset.update((prev) => ({ ...prev, filterGroup }));
    }
  }

  public addChild(
    id: string,
    addType: FormKind,
    obj?: { index: number; item: Readonly<FormGroup | FormField> },
  ): void {
    const filterSet: Readonly<FilterFormSet> = this.#formset.get();
    const filterGroup = filterSet.filterGroup;
    const traverse = (form: FormGroup | FormField): void => {
      if (form.id === id && form.kind === FormKind.Group) {
        if (obj) {
          form.children.splice(obj.index, 0, structuredClone(obj.item));
        } else {
          form.children.push(addType === FormKind.Group ? getInitGroup() : getInitField());
        }
        return;
      }

      if (form.kind === FormKind.Group) {
        for (const child of form.children) {
          traverse(child);
        }
      }
    };

    traverse(filterGroup);
    this.#formset.update((prev) => ({ ...prev, filterGroup }));
  }

  public removeChild(id: string): void {
    const filterSet: Readonly<FilterFormSet> = this.#formset.get();
    const filterGroup = filterSet.filterGroup;

    if (filterGroup.id === id) {
      // if remove top group
      this.#formset.set({ ...structuredClone(INIT_FORMSET), showArchived: filterSet.showArchived });
      return;
    }

    const traverse = (form: FormGroup | FormField): void => {
      if (form.kind === FormKind.Group) {
        const prevLength = form.children.length;
        form.children = form.children.filter((c) => c.id !== id);
        if (prevLength === form.children.length) {
          for (const child of form.children) {
            traverse(child);
          }
        }
      }
    };
    traverse(filterGroup);
    this.#formset.update((prev) => ({ ...prev, filterGroup }));
  }

  public setArchivedValue(val: boolean): void {
    const filterSet: Readonly<FilterFormSet> = this.#formset.get();
    this.#formset.set({ ...filterSet, showArchived: val });
  }
}
