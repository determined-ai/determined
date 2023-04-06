import { Observable, observable } from 'micro-observables';

import { FilterFormSet, FormField, FormGroup, Operator } from './type';

const INIT_FORMSET: FilterFormSet = {
  filterSet: { children: [], conjunction: 'and', id: 'ROOT', type: 'group' }, // default
};

export class FormClassStore {
  #formset = observable<FilterFormSet>(INIT_FORMSET);

  constructor(data?: FilterFormSet) {
    if (data) {
      this.#formset = observable<FilterFormSet>(data);
    }
  }

  public get formset(): Observable<Readonly<FilterFormSet>> {
    return this.#formset.readOnly();
  }

  public setFieldValue(
    id: string,
    keyType:
      | keyof Pick<FormField, 'columnName' | 'operator' | 'value'>
      | keyof Pick<FormGroup, 'conjunction'>,
    value: string | string[] | number | number[],
  ): void {
    const set = this.#formset.get().filterSet;
    const recur = (form: FormGroup | FormField): FormGroup | FormField | undefined => {
      if (form.id === id) {
        return form;
      }
      if (form.type === 'group' && form.children.length === 0) {
        return undefined;
      }

      if (form.type === 'group') {
        for (const child of form.children) {
          const ans = recur(child);
          if (ans) {
            return ans;
          }
        }
      }
      return undefined;
    };

    const ans = recur(set);

    if (ans) {
      if (ans.type === 'field') {
        if (keyType === 'columnName' && typeof value === 'string') {
          ans.columnName = value;
        } else if (keyType === 'operator' && typeof value === 'string') {
          ans.operator = value as Operator;
        } else if (keyType === 'value') {
          ans.value = value;
        }
      } else if (ans.type === 'group') {
        if (keyType === 'conjunction' && (value === 'and' || value === 'or')) {
          ans.conjunction = value;
        }
      }
      this.#formset.set({ filterSet: set });
    }
  }

  public addChild(
    id: string,
    addType: 'group' | 'field',
    obj?: Readonly<FormGroup | FormField>,
  ): void {
    const set = this.#formset.get().filterSet;
    const recur = (form: FormGroup | FormField): void => {
      if (form.id === id && form.type === 'group') {
        if (obj) {
          form.children.push(obj);
          return;
        }
        form.children.push(
          addType === 'group'
            ? {
                children: [],
                conjunction: 'and',
                id: crypto.randomUUID(),
                type: 'group',
              }
            : {
                columnName: 'id',
                id: crypto.randomUUID(),
                operator: 'contains',
                type: 'field',
                value: undefined,
              },
        );
        return;
      }
      if (form.type === 'group' && form.children.length === 0) {
        return;
      }

      if (form.type === 'group') {
        for (const child of form.children) {
          recur(child);
        }
      }
    };

    recur(set);
    this.#formset.set({ filterSet: set });
  }

  public removeChild(id: string): void {
    const set = this.#formset.get().filterSet;

    if (set.id === id) {
      this.#formset.set(structuredClone(INIT_FORMSET));
      return;
    }

    const recur = (form: FormGroup | FormField): void => {
      if (form.type === 'group') {
        form.children = form.children.filter((c) => c.id !== id);
        for (const child of form.children) {
          recur(child);
        }
      }
    };
    recur(set);
    this.#formset.set({ filterSet: set });
  }
}

export const formSets: FilterFormSet = {
  filterSet: {
    children: [
      {
        columnName: 'tags',
        id: 'level1',
        operator: 'contains',
        type: 'field',
        value: 'test',
      },
      {
        children: [
          {
            columnName: 'user',
            id: 'stringdsdff123',
            operator: 'eq',
            type: 'field',
            value: 1,
          },
          {
            columnName: 'state',
            id: 'stringdsdff3',
            operator: 'notEmpty',
            type: 'field',
            value: 'test',
          },
        ],
        conjunction: 'or',
        id: 'sdsdff',
        type: 'group',
      },
      {
        columnName: 'name',
        id: 'stringdf123',
        operator: 'contains',
        type: 'field',
        value: 'test',
      },
      {
        columnName: 'id',
        id: 'gsstringdfs123',
        operator: 'greaterEq',
        type: 'field',
        value: 'test',
      },
    ],
    conjunction: 'and',
    id: 'ROOT',
    type: 'group',
  },
};
