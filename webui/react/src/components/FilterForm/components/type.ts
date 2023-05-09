import { V1ColumnType, V1LocationType, V1ProjectColumn } from 'services/api-ts-sdk';
import { ValueOf } from 'shared/types';
import { RunState } from 'types';

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

type FormFieldWithoutId = Omit<FormField, 'id'>;

type FormGroupWithoutId = {
  readonly kind: typeof FormKind.Group;
  conjunction: Conjunction;
  children: (FormGroupWithoutId | FormFieldWithoutId)[];
};

export type FilterFormSetWithoutId = {
  filterGroup: FormGroupWithoutId;
  showArchived: boolean;
};

export const Conjunction = {
  And: 'and',
  Or: 'or',
} as const;

export type Conjunction = ValueOf<typeof Conjunction>;

export const Operator = {
  Contains: 'contains',
  Eq: '=',
  Greater: '>',
  GreaterEq: '>=',
  IsEmpty: 'isEmpty',
  Less: '<',
  LessEq: '<=',
  NotContains: 'notContains',
  NotEmpty: 'notEmpty',
  NotEq: '!=',
} as const;

export type Operator = ValueOf<typeof Operator>;

export const ReadableOperator: Record<Operator, string> = {
  [Operator.Contains]: 'contains',
  [Operator.Eq]: '=',
  [Operator.Greater]: '>',
  [Operator.GreaterEq]: '>=',
  [Operator.IsEmpty]: 'is empty',
  [Operator.Less]: '<',
  [Operator.LessEq]: '<=',
  [Operator.NotContains]: 'not contains',
  [Operator.NotEmpty]: 'not empty',
  [Operator.NotEq]: '!=',
} as const;

export const AvailableOperators = {
  [V1ColumnType.NUMBER]: [
    Operator.Eq,
    Operator.NotEq,
    Operator.Greater,
    Operator.GreaterEq,
    Operator.Less,
    Operator.LessEq,
  ],
  [V1ColumnType.TEXT]: [
    Operator.Contains,
    Operator.NotContains,
    Operator.Eq,
    Operator.NotEq,
    Operator.IsEmpty,
    Operator.NotEmpty,
  ],
  [V1ColumnType.DATE]: [
    // No Eq and NotEq for date because date should be used with range
    Operator.Greater,
    Operator.GreaterEq,
    Operator.Less,
    Operator.LessEq,
  ],
  [V1ColumnType.UNSPECIFIED]: Object.values(Operator), // show all of operators
} as const;

export const RUN_STATES = [
  RunState.Active,
  RunState.Paused,
  RunState.Canceled,
  RunState.Completed,
  RunState.Error,
] as const;

export const SEARCHER_TYPE = ['adaptive_asha', 'single', 'random'] as const;

export const SpecialColumnNames = ['user', 'state', 'resourcePool', 'searcherType'] as const;

export type SpecialColumnNames = (typeof SpecialColumnNames)[number];
