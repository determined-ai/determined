import * as t from 'io-ts';

import { ioColumnType, ioLocationType } from 'ioTypes';
import { V1ColumnType } from 'services/api-ts-sdk';
import { RunState, ValueOf } from 'types';

export const FormKind = {
  Field: 'field',
  Group: 'group',
} as const;

export type FormKind = ValueOf<typeof FormKind>;

export type FormFieldValue = string | number | null;

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

const READABLE_TEXT_OPERATOR: Record<Operator, string> = {
  [Operator.Contains]: 'contains',
  [Operator.NotContains]: 'does not contain',
  [Operator.Eq]: 'is',
  [Operator.NotEq]: 'is not',
  [Operator.Greater]: '>',
  [Operator.GreaterEq]: '>=',
  [Operator.Less]: '<',
  [Operator.LessEq]: '<=',
  [Operator.IsEmpty]: 'is empty',
  [Operator.NotEmpty]: 'is not empty',
} as const;

const READABLE_NUMBER_OPERATOR: Record<Operator, string> = {
  [Operator.Contains]: 'contains',
  [Operator.NotContains]: 'does not contain',
  [Operator.Eq]: '=',
  [Operator.NotEq]: '!=',
  [Operator.Greater]: '>',
  [Operator.GreaterEq]: '>=',
  [Operator.Less]: '<',
  [Operator.LessEq]: '<=',
  [Operator.IsEmpty]: 'is empty',
  [Operator.NotEmpty]: 'is not empty',
} as const;

const READABLE_DATE_OPERATOR: Record<Operator, string> = {
  [Operator.Contains]: 'contains',
  [Operator.NotContains]: 'does not contain',
  [Operator.Eq]: 'on',
  [Operator.NotEq]: 'not on',
  [Operator.Greater]: 'after',
  [Operator.GreaterEq]: 'on or after',
  [Operator.Less]: 'before',
  [Operator.LessEq]: 'on or before',
  [Operator.IsEmpty]: 'is empty',
  [Operator.NotEmpty]: 'is not empty',
} as const;

export const ReadableOperator: Record<V1ColumnType, Record<Operator, string>> = {
  [V1ColumnType.TEXT]: READABLE_TEXT_OPERATOR,
  [V1ColumnType.NUMBER]: READABLE_NUMBER_OPERATOR,
  [V1ColumnType.DATE]: READABLE_DATE_OPERATOR,
  [V1ColumnType.UNSPECIFIED]: READABLE_TEXT_OPERATOR,
} as const;

export const AvailableOperators = {
  [V1ColumnType.TEXT]: [
    Operator.Contains,
    Operator.NotContains,
    Operator.Eq,
    Operator.NotEq,
    Operator.IsEmpty,
    Operator.NotEmpty,
  ],
  [V1ColumnType.NUMBER]: [
    Operator.Eq,
    Operator.NotEq,
    Operator.Greater,
    Operator.GreaterEq,
    Operator.Less,
    Operator.LessEq,
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

export const SEARCHER_TYPE = ['adaptive_asha', 'single', 'grid', 'random', 'custom'] as const;

export const SpecialColumnNames = ['user', 'state', 'resourcePool', 'searcherType'] as const;

export type SpecialColumnNames = (typeof SpecialColumnNames)[number];

const IOOperator: t.Type<Operator> = t.keyof({
  [Operator.Contains]: null,
  [Operator.Eq]: null,
  [Operator.Greater]: null,
  [Operator.GreaterEq]: null,
  [Operator.IsEmpty]: null,
  [Operator.Less]: null,
  [Operator.LessEq]: null,
  [Operator.NotContains]: null,
  [Operator.NotEmpty]: null,
  [Operator.NotEq]: null,
});

const FormField = t.type({
  columnName: t.string,
  id: t.readonly(t.string),
  kind: t.readonly(t.literal(FormKind.Field)),
  location: ioLocationType,
  operator: IOOperator,
  type: ioColumnType,
  value: t.union([t.string, t.number, t.null]),
});

export type FormField = t.TypeOf<typeof FormField>;

const IOFormGroup: t.Type<FormGroup> = t.recursion('IOFormGroup', () =>
  t.type({
    children: t.array(t.union([FormField, IOFormGroup])),
    conjunction: IOConjunction,
    id: t.readonly(t.string),
    kind: t.readonly(t.literal(FormKind.Group)),
  }),
);

const IOConjunction = t.union([t.literal('and'), t.literal('or')]);

export const IOFilterFormSet = t.type({
  filterGroup: IOFormGroup,
  showArchived: t.boolean,
});
