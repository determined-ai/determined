import { V1ColumnType, V1LocationType, V1ProjectColumn } from 'services/api-ts-sdk';
import { ValueOf } from 'shared/types';

export const FormKind = {
  Field: 'field',
  Group: 'group',
} as const;

export type FormKind = ValueOf<typeof FormKind>;

export type FormFieldValue = string | number | null;

export type FormField = {
  readonly id: string;
  readonly kind: typeof FormKind.Field;
  columnName: V1ProjectColumn['column'];
  location: V1LocationType;
  type: V1ColumnType;
  operator: Operator;
  value: FormFieldValue;
};

export type FormGroup = {
  readonly id: string;
  readonly kind: typeof FormKind.Group;
  conjunction: Conjunction;
  children: (FormGroup | FormField)[];
};

export type FilterFormSet = {
  filterGroup: FormGroup;
  showArchived: boolean;
};

export const Conjunction = {
  And: 'and',
  Or: 'or',
} as const;

export type Conjunction = ValueOf<typeof Conjunction>;

export const Operator = {
  contains: 'contains',
  eq: '=',
  greater: '>',
  greaterEq: '>=',
  is: 'is',
  isEmpty: 'is empty',
  isNot: 'is not',
  less: '<',
  lessEq: '<=',
  notContain: 'not contains',
  notEmpty: 'not empty',
  notEq: '!=',
} as const;

export type Operator = ValueOf<typeof Operator>;

export const AvaliableOperators = {
  [V1ColumnType.NUMBER]: [
    Operator.eq,
    Operator.notEq,
    Operator.greater,
    Operator.greaterEq,
    Operator.less,
    Operator.lessEq,
  ],
  [V1ColumnType.TEXT]: [
    Operator.contains,
    Operator.notContain,
    Operator.isEmpty,
    Operator.notEmpty,
    Operator.is,
    Operator.isNot,
  ],
  [V1ColumnType.DATE]: [
    // no Operator.eq for date because date should be used with range
    Operator.notEq,
    Operator.greater,
    Operator.greaterEq,
    Operator.less,
    Operator.lessEq,
  ],
  [V1ColumnType.UNSPECIFIED]: Object.values(Operator), // show all of operators
} as const;
