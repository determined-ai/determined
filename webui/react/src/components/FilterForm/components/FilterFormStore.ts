import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
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
  MatchFunc,
  Operator,
  PatchFunc,
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

export const getInitField = (): FormField => ({
  columnName: 'name',
  id: uuidv4(),
  kind: FormKind.Field,
  location: V1LocationType.EXPERIMENT,
  operator: AvailableOperators[V1ColumnType.TEXT][0],
  type: V1ColumnType.TEXT,
  value: null,
});

const isNotUndefined = <T>(arg: T): arg is Exclude<T, undefined> => arg !== undefined;

export class FilterFormStore {
  #formset: WritableObservable<Loadable<FilterFormSet>> = observable(NotLoaded);

  public init(data?: Readonly<FilterFormSet>): void {
    this.#formset.update(() => Loaded(structuredClone(data ? data : INIT_FORMSET)));
  }

  public get formset(): Observable<Loadable<FilterFormSet>> {
    return this.#formset.readOnly();
  }

  public get asJsonString(): Observable<string> {
    const replacer = (key: string, value: unknown): unknown => {
      return key === 'id' ? undefined : value;
    };
    return this.#formset.select((loadableFormset) =>
      Loadable.match(loadableFormset, {
        _: () => '',
        Loaded: (formset) => {
          const sweepedForm = this.#sweepInvalid(structuredClone(formset.filterGroup));
          const newFormSet: FilterFormSetWithoutId = JSON.parse(
            JSON.stringify({ ...formset, filterGroup: sweepedForm }, replacer),
          );
          return JSON.stringify(newFormSet);
        },
      }),
    );
  }

  public get fieldCount(): Observable<number> {
    return this.getFieldCount();
  }

  public getFieldCount(field?: string): Observable<number> {
    const countFields = (form: Readonly<FormGroup>): number => {
      let count = 0;
      for (const child of form.children) {
        count +=
          child.kind === FormKind.Group
            ? countFields(child)
            : !field || field === child.columnName
              ? 1
              : 0;
      }
      return count;
    };
    return this.#formset.select((loadableFormset) =>
      Loadable.match(loadableFormset, {
        _: () => 0,
        Loaded: (formset) => {
          const validFilterGroup = this.#sweepInvalid(formset.filterGroup);
          return countFields(validFilterGroup);
        },
      }),
    );
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
    this.#formset.update((loadablePrev) =>
      Loadable.map(loadablePrev, (prev) => {
        const filterGroup = this.#sweepInvalid(prev.filterGroup);
        return { ...prev, filterGroup };
      }),
    );
  }

  #updateForm(
    match: MatchFunc<FormField | FormGroup>,
    patch: PatchFunc<FormField | FormGroup>,
    autoSweep?: boolean,
  ): void {
    this.#formset.update((loadableFilterSet) => {
      return Loadable.map(loadableFilterSet, (filterSet) => {
        // keep updates to a minimum -- only return a new filterset
        // if a change occurred
        let hit = false;
        const traverse = (entity: FormGroup | FormField): FormGroup | FormField | undefined => {
          if (match(entity)) {
            const retVal = patch(entity);
            if (entity !== retVal) {
              hit = true;
              return retVal;
            }
            return entity;
          }
          if (entity.kind === FormKind.Group) {
            const children = entity.children.map(traverse).filter(isNotUndefined);

            return {
              ...entity,
              children,
            };
          }
          return entity;
        };

        const filterGroup =
          traverse(filterSet.filterGroup) || structuredClone(INIT_FORMSET).filterGroup;
        if (!hit) {
          return filterSet;
        }
        if (filterGroup.kind === FormKind.Field) {
          throw new Error('patch changed base filter group to field');
        }
        return {
          ...filterSet,
          filterGroup: autoSweep ? this.#sweepInvalid(filterGroup) : filterGroup,
        };
      });
    });
  }

  #updateField(id: string, patch: PatchFunc<FormField>): void {
    return this.#updateForm(
      (field) => field.id === id,
      (arg) => {
        if (arg.kind === FormKind.Group) {
          return arg;
        }
        return patch(arg);
      },
    );
  }

  #updateGroup(id: string, patch: PatchFunc<FormGroup>): void {
    return this.#updateForm(
      (field) => field.id === id,
      (arg) => {
        if (arg.kind === FormKind.Field) {
          return arg;
        }
        return patch(arg);
      },
    );
  }

  public setFieldColumnName(
    id: string,
    col: Pick<V1ProjectColumn, 'location' | 'type' | 'column'>,
  ): void {
    return this.#updateField(id, (form) => {
      if (form.columnName === col.column && form.location === col.location) {
        return form;
      }
      return {
        ...form,
        columnName: col.column,
        location: col.location,
        type: col.type,
      };
    });
  }

  public setFieldOperator(id: string, operator: Operator): void {
    return this.#updateField(id, (form) =>
      form.operator === operator ? form : { ...form, operator },
    );
  }

  public setFieldConjunction(id: string, conjunction: Conjunction): void {
    return this.#updateGroup(id, (form) =>
      form.conjunction === conjunction ? form : { ...form, conjunction },
    );
  }

  public setFieldValue(id: string, value: FormFieldValue): void {
    return this.#updateField(id, (form) => (form.value === value ? form : { ...form, value }));
  }

  public addChild(
    id: string,
    addType: FormKind,
    obj?: { index: number; item: Readonly<FormGroup | FormField> },
  ): void {
    return this.#updateGroup(id, (form) => {
      const children = obj
        ? form.children
            .slice(0, obj.index)
            .concat([structuredClone(obj.item)], form.children.slice(obj.index))
        : [...form.children, addType === FormKind.Group ? getInitGroup() : getInitField()];
      return {
        ...form,
        children,
      };
    });
  }

  public removeByField(column: string): void {
    this.#updateForm(
      (field) => field.kind === FormKind.Field && field.columnName === column,
      () => undefined,
      true,
    );
  }

  public removeChild(id: string): void {
    this.#updateForm(
      (field) => field.id === id,
      () => undefined,
    );
  }

  public setArchivedValue(val: boolean): void {
    this.#formset.update((loadableFilterSet) => {
      return Loadable.map(loadableFilterSet, (fs) => ({ ...fs, showArchived: val }));
    });
  }
}
