import { ExperimentColumnName } from 'pages/ExperimentList.settings';
import { ValueOf } from 'shared/types';

export type ExperimentFilterColumnName = Exclude<ExperimentColumnName, 'action' | 'archived'>;

export const FormKind = {
  Field: 'field',
  Group: 'group',
} as const;

export type FormKind = ValueOf<typeof FormKind>;

export type FormFieldValue = string | number | null;

export type FormField = {
  readonly id: string;
  readonly kind: typeof FormKind.Field;
  columnName: ExperimentFilterColumnName;
  operator: Operator;
  value: FormFieldValue;
};

export type FormGroup = {
  readonly id: string;
  readonly kind: typeof FormKind.Group;
  conjunction: Conjunction;
  children: (FormGroup | FormField)[];
};

export type KeyType =
  | keyof Pick<FormField, 'columnName' | 'operator' | 'value'>
  | keyof Pick<FormGroup, 'conjunction'>;

export type FilterFormSet = {
  filterGroup: FormGroup;
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
  number: [
    Operator.eq,
    Operator.notEq,
    Operator.greater,
    Operator.greaterEq,
    Operator.less,
    Operator.lessEq,
  ],
  string: [
    Operator.contains,
    Operator.notContain,
    Operator.isEmpty,
    Operator.notEmpty,
    Operator.is,
    Operator.isNot,
  ],
} as const;

export const ColumnType: Record<ExperimentFilterColumnName, keyof typeof AvaliableOperators> = {
  checkpointCount: 'number',
  checkpointSize: 'number',
  description: 'string',
  duration: 'number',
  forkedFrom: 'number',
  id: 'number',
  name: 'string',
  numTrials: 'number',
  progress: 'number',
  resourcePool: 'number',
  searcherMetricValue: 'number',
  searcherType: 'string',
  startTime: 'number',
  state: 'string',
  tags: 'string',
  user: 'number',
} as const;
