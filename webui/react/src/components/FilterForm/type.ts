import { ValueOf } from 'shared/types';

export const FormType = {
  Field: 'field',
  Group: 'group',
} as const;

export type FormType = ValueOf<typeof FormType>;

export type FormFieldValue = string | number | null;

export type FormField = {
  readonly id: string;
  readonly type: typeof FormType.Field;
  columnName: string;
  operator: Operator;
  value: FormFieldValue;
};

export const Conjunction = {
  And: 'and',
  Or: 'or',
} as const;

export type Conjunction = ValueOf<typeof Conjunction>;

export type FormGroup = {
  readonly id: string;
  readonly type: typeof FormType.Group;
  conjunction: Conjunction;
  children: (FormGroup | FormField)[];
};

export type FilterFormSet = {
  filterGroup: FormGroup;
};

export const OperatorMap = {
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

export type Operator = ValueOf<typeof OperatorMap>;

export const AvaliableOperators = {
  number: [
    OperatorMap.eq,
    OperatorMap.notEq,
    OperatorMap.greater,
    OperatorMap.greaterEq,
    OperatorMap.less,
    OperatorMap.lessEq,
  ],
  string: [
    OperatorMap.contains,
    OperatorMap.notContain,
    OperatorMap.isEmpty,
    OperatorMap.notEmpty,
    OperatorMap.is,
    OperatorMap.isNot,
  ],
} as const;

export const ColumnType: Record<string, keyof typeof AvaliableOperators> = {
  id: 'number',
  name: 'string',
  state: 'string',
  tags: 'string',
  user: 'number',
} as const;
