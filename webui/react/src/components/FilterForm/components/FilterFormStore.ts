import { Observable, observable } from 'micro-observables';

import {
  Conjunction,
  ExperimentFilterColumnName,
  FilterFormSet,
  FormField,
  FormFieldValue,
  FormGroup,
  FormKind,
  KeyType,
  Operator,
} from './type';

export const ITEM_LIMIT = 50;

const INIT_FORMSET: Readonly<FilterFormSet> = {
  filterGroup: { children: [], conjunction: Conjunction.And, id: 'ROOT', kind: FormKind.Group },
};

const getInitGroup = (): FormGroup => ({
  children: [],
  conjunction: Conjunction.And,
  id: crypto.randomUUID(),
  kind: FormKind.Group,
});

const getInitField = (): FormField => ({
  columnName: 'id',
  id: crypto.randomUUID(),
  kind: FormKind.Field,
  operator: '=',
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
    const formGroup: Readonly<FilterFormSet> = this.#formset.get();
    return formGroup;
  }

  public setFieldValue(id: string, keyType: KeyType, value: FormFieldValue): void {
    const filterGroup = this.#formset.get().filterGroup;
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

    const ans = recur(filterGroup);
    if (ans) {
      if (ans.kind === FormKind.Field) {
        if (keyType === 'columnName' && typeof value === 'string') {
          ans.columnName = value as ExperimentFilterColumnName;
        } else if (keyType === 'operator' && typeof value === 'string') {
          ans.operator = value as Operator;
        } else if (keyType === 'value') {
          ans.value = value;
        }
      } else if (ans.kind === FormKind.Group) {
        if (keyType === 'conjunction' && (value === Conjunction.And || value === Conjunction.Or)) {
          ans.conjunction = value;
        }
      }
      this.#formset.set({ filterGroup });
    }
  }

  public addChild(
    id: string,
    addType: FormKind,
    index: number,
    obj?: Readonly<FormGroup | FormField>,
  ): void {
    const filterGroup = this.#formset.get().filterGroup;
    const recur = (form: FormGroup | FormField): void => {
      if (form.id === id && form.kind === FormKind.Group) {
        if (obj) {
          form.children.splice(index, 0, obj);
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
    this.#formset.set({ filterGroup });
  }

  public removeChild(id: string): void {
    const filterGroup = this.#formset.get().filterGroup;

    if (filterGroup.id === id) {
      // if remove top group
      this.#formset.set(structuredClone(INIT_FORMSET));
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
    this.#formset.set({ filterGroup });
  }
}
