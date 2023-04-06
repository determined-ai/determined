export const ItemTypes = {
  FIELD: 'field',
  GROUP: 'group',
} as const;

export type FormField = {
  readonly id: string;
  readonly type: typeof ItemTypes.FIELD;
  columnName: string;
  operator: Operator;
  value: string | string[] | number | number[] | undefined;
};

export type Conjunction = 'and' | 'or';

export type FormGroup = {
  readonly id: string;
  readonly type: typeof ItemTypes.GROUP;
  conjunction: Conjunction;
  children: (FormGroup | FormField)[];
};

export type FilterFormSet = {
  filterSet: FormGroup;
};

export type Operator =
  | 'contains'
  | 'in'
  | 'is'
  | 'eq'
  | 'greater'
  | 'greaterEq'
  | 'isEmpty'
  | 'isNot'
  | 'less'
  | 'lessEq'
  | 'notContain'
  | 'notEmpty'
  | 'notEq'
  | 'notIn';

export const OperatorMap: Record<Operator, string> = {
  contains: 'contains',
  eq: '=',
  greater: '>',
  greaterEq: '>=',
  in: 'in',
  is: 'is',
  isEmpty: 'is empty',
  isNot: 'is not',
  less: '<',
  lessEq: '<=',
  notContain: 'not contains',
  notEmpty: 'not empty',
  notEq: '!=',
  notIn: 'not in',
} as const;
