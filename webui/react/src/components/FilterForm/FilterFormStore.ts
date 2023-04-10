import { Observable, observable } from 'micro-observables';

import {
  ColumnType,
  Conjunction,
  ExperimentFilterColumnName,
  FilterFormSet,
  FormField,
  FormFieldValue,
  FormGroup,
  FormType,
  KeyType,
  Operator,
} from './type';

const INIT_FORMSET: FilterFormSet = {
  filterGroup: { children: [], conjunction: Conjunction.And, id: 'ROOT', type: FormType.Group },
};

const getInitGroup = (): FormGroup => ({
  children: [],
  conjunction: Conjunction.And,
  id: crypto.randomUUID(),
  type: FormType.Group,
});

const getInitField = (): FormField => ({
  columnName: 'id',
  id: crypto.randomUUID(),
  operator: '=',
  type: FormType.Field,
  value: null,
});

const OperatorQueryMap: Record<Operator, (colName: string, val: FormFieldValue) => string> = {
  '!=': (colName: string, val: FormFieldValue) => `-${colName}:${val}`,
  '<': (colName: string, val: FormFieldValue) => `${colName}<${val}`,
  '<=': (colName: string, val: FormFieldValue) => `-${colName}<=${val}`,
  '=': (colName: string, val: FormFieldValue) => `${colName}:${val}`,
  '>': (colName: string, val: FormFieldValue) => `${colName}>${val}`,
  '>=': (colName: string, val: FormFieldValue) => `${colName}>=${val}`,
  'contains': (colName: string, val: FormFieldValue) => `${colName}~${val}`,
  'is': (colName: string, val: FormFieldValue) => `${colName}:${val}`,
  'is empty': (colName: string) => `${colName}:null`,
  'is not': (colName: string, val: FormFieldValue) => `-${colName}: ${val}`,
  'not contains': (colName: string, val: FormFieldValue) => `-${colName}~${val}`,
  'not empty': (colName: string) => `-${colName}:null`,
} as const;

export class FilterFormStore {
  #formset = observable<FilterFormSet>(INIT_FORMSET);

  constructor(data?: FilterFormSet) {
    if (data) {
      this.#formset = observable<FilterFormSet>(data);
    }
  }

  public get formset(): Observable<Readonly<FilterFormSet>> {
    return this.#formset.readOnly();
  }

  public get query(): string {
    const formGroup: Readonly<FormGroup> = this.#formset.get().filterGroup;

    const recur = (form: FormGroup | FormField): string => {
      if (form.type === 'field') {
        const type = ColumnType[form.columnName];
        const escapeVal = form.value?.toString().replaceAll('"', '\\"') ?? null;
        const value = type === 'string' && escapeVal != null ? `"${escapeVal}"` : escapeVal;
        const func = OperatorQueryMap[form.operator];
        return func(form.columnName, value);
      }
      const arr = [];
      if (form.type === 'group') {
        for (const child of form.children) {
          const ans = recur(child);
          arr.push(ans);
        }
      }
      return `(${arr.join(` ${form.conjunction} `.toUpperCase())})`;
    };
    return recur(formGroup);
  }

  public setFieldValue(id: string, keyType: KeyType, value: FormFieldValue): void {
    const filterGroup = this.#formset.get().filterGroup;
    const recur = (form: FormGroup | FormField): FormGroup | FormField | undefined => {
      if (form.id === id) {
        return form;
      }
      if (form.type === FormType.Group && form.children.length === 0) {
        return undefined;
      }

      if (form.type === FormType.Group) {
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
      if (ans.type === FormType.Field) {
        if (keyType === 'columnName' && typeof value === 'string') {
          ans.columnName = value as ExperimentFilterColumnName;
        } else if (keyType === 'operator' && typeof value === 'string') {
          ans.operator = value as Operator;
        } else if (keyType === 'value') {
          ans.value = value;
        }
      } else if (ans.type === FormType.Group) {
        if (keyType === 'conjunction' && (value === Conjunction.And || value === Conjunction.Or)) {
          ans.conjunction = value;
        }
      }
      this.#formset.set({ filterGroup });
    }
  }

  public addChild(
    id: string,
    addType: FormType,
    index: number,
    obj?: Readonly<FormGroup | FormField>,
  ): void {
    const filterGroup = this.#formset.get().filterGroup;
    const recur = (form: FormGroup | FormField): void => {
      if (form.id === id && form.type === FormType.Group) {
        if (obj) {
          form.children.splice(index, 0, obj);
        } else {
          form.children.push(addType === FormType.Group ? getInitGroup() : getInitField());
        }
        return;
      }

      if (form.type === FormType.Group) {
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
      this.#formset.set(structuredClone(INIT_FORMSET));
      return;
    }

    const recur = (form: FormGroup | FormField): void => {
      if (form.type === FormType.Group) {
        form.children = form.children.filter((c) => c.id !== id);
        for (const child of form.children) {
          recur(child);
        }
      }
    };
    recur(filterGroup);
    this.#formset.set({ filterGroup });
  }
}

export const formSets: FilterFormSet = {
  filterGroup: {
    children: [
      {
        columnName: 'tags',
        id: 'level1',
        operator: 'contains',
        type: FormType.Field,
        value: 'test',
      },
      {
        children: [
          {
            columnName: 'user',
            id: 'stringdsdff123',
            operator: '=',
            type: FormType.Field,
            value: '1',
          },
          {
            columnName: 'state',
            id: 'stringdsdff3',
            operator: '!=',
            type: FormType.Field,
            value: 'test',
          },
        ],
        conjunction: 'or',
        id: 'sdsdff',
        type: FormType.Group,
      },
      {
        columnName: 'name',
        id: 'stringdf123',
        operator: 'contains',
        type: FormType.Field,
        value: 'test',
      },
      {
        columnName: 'id',
        id: 'gsstringdfs123',
        operator: '>=',
        type: FormType.Field,
        value: '1',
      },
    ],
    conjunction: Conjunction.And,
    id: 'ROOT',
    type: FormType.Group,
  },
};
