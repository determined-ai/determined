import { ExperimentColumnName } from 'pages/ExperimentList.settings';
import { V1ColumnType } from 'services/api-ts-sdk/api';
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
    Operator.eq,
    Operator.notEq,
    Operator.greater,
    Operator.greaterEq,
    Operator.less,
    Operator.lessEq,
  ],
} as const;

export const ColumnType: Record<ExperimentFilterColumnName, keyof typeof AvaliableOperators> = {
  checkpointCount: V1ColumnType.NUMBER,
  checkpointSize: V1ColumnType.NUMBER,
  description: V1ColumnType.TEXT,
  duration: V1ColumnType.NUMBER,
  forkedFrom: V1ColumnType.NUMBER,
  id: V1ColumnType.NUMBER,
  name: V1ColumnType.TEXT,
  numTrials: V1ColumnType.NUMBER,
  progress: V1ColumnType.NUMBER,
  resourcePool: V1ColumnType.NUMBER,
  searcherMetricValue: V1ColumnType.NUMBER,
  searcherType: V1ColumnType.TEXT,
  startTime: V1ColumnType.DATE,
  state: V1ColumnType.TEXT,
  tags: V1ColumnType.TEXT,
  user: V1ColumnType.NUMBER,
} as const;
