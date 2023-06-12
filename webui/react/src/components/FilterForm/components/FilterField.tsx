import { HolderOutlined } from '@ant-design/icons';
import { DatePicker } from 'antd';
import type { SelectProps as AntdSelectProps } from 'antd';
import type { DatePickerProps } from 'antd/es/date-picker';
import dayjs from 'dayjs';
import { useObservable } from 'micro-observables';
import { useCallback, useState } from 'react';
import { useDrag, useDrop } from 'react-dnd';
import { debounce } from 'throttle-debounce';

import ConjunctionContainer from 'components/FilterForm/components/ConjunctionContainer';
import { FilterFormStore, getInitField } from 'components/FilterForm/components/FilterFormStore';
import {
  AvailableOperators,
  Conjunction,
  FormField,
  FormFieldValue,
  FormGroup,
  FormKind,
  Operator,
  ReadableOperator,
  RUN_STATES,
  SEARCHER_TYPE,
  SpecialColumnNames,
} from 'components/FilterForm/components/type';
import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import Input from 'components/kit/Input';
import InputNumber from 'components/kit/InputNumber';
import Select, { SelectValue } from 'components/kit/Select';
import { V1ColumnType, V1ProjectColumn } from 'services/api-ts-sdk';
import clusterStore from 'stores/cluster';
import userStore from 'stores/users';
import { Loadable } from 'utils/loadable';
import { alphaNumericSorter } from 'utils/sort';

import css from './FilterField.module.scss';

const debounceFunc = debounce(1000, (func: () => void) => {
  func();
});

interface Props {
  index: number; // start from 0
  field: FormField;
  parentId: string;
  conjunction: Conjunction;
  formStore: FilterFormStore;
  level: number; // start from 0
  columns: V1ProjectColumn[];
}

const FilterField = ({
  field,
  conjunction,
  formStore,
  index,
  parentId,
  level,
  columns,
}: Props): JSX.Element => {
  const users = Loadable.getOrElse([], useObservable(userStore.getUsers()));
  const resourcePools = Loadable.getOrElse([], useObservable(clusterStore.resourcePools));

  const currentColumn = columns.find((c) => c.column === field.columnName);
  const isSpecialColumn = (SpecialColumnNames as ReadonlyArray<string>).includes(field.columnName);

  const [fieldValue, setFieldValue] = useState<FormFieldValue>(field.value);

  // use this function to update field value
  const updateFieldValue = (fieldId: string, value: FormFieldValue, debounceUpdate = false) => {
    if (debounceUpdate) {
      debounceFunc(() => formStore.setFieldValue(fieldId, value));
    } else {
      formStore.setFieldValue(fieldId, value);
    }
    setFieldValue(value);
  };

  const onChangeColumnName = (value: SelectValue) => {
    const prevType = currentColumn?.type;
    const newCol = columns.find((c) => c.column === value?.toString() ?? '');
    if (newCol) {
      formStore.setFieldColumnName(field.id, newCol);

      if ((SpecialColumnNames as ReadonlyArray<string>).includes(field.columnName)) {
        formStore.setFieldOperator(field.id, Operator.Eq);
        updateFieldValue(field.id, null);
      } else if (prevType !== newCol?.type) {
        const defaultOperator: Operator =
          AvailableOperators[newCol?.type ?? V1ColumnType.UNSPECIFIED][0];
        formStore.setFieldOperator(field.id, defaultOperator);
        updateFieldValue(field.id, null);
      }
    }
  };

  const getSpecialOptions = (columnName: SpecialColumnNames): AntdSelectProps['options'] => {
    switch (columnName) {
      case 'resourcePool':
        return resourcePools.map((rp) => ({ label: rp.name, value: rp.name }));
      case 'state':
        return RUN_STATES.map((state) => ({ label: state, value: state }));
      case 'searcherType':
        return SEARCHER_TYPE.map((searcher) => ({ label: searcher, value: searcher }));
      case 'user':
        return users
          .sort((a, b) => alphaNumericSorter(a.username, b.username))
          .map((user) => ({ label: user.username, value: user.username }));
      default:
        // eslint-disable-next-line no-case-declarations, @typescript-eslint/no-unused-vars
        const _exhaustiveCheck: never = columnName;
        throw new Error(`${columnName} is not columnName.`);
    }
  };

  const [, drag, preview] = useDrag<{ form: FormField; index: number }, unknown, unknown>(
    () => ({
      item: { form: field, index },
      type: FormKind.Field,
    }),
    [field],
  );

  const [{ isOverCurrent, canDrop }, drop] = useDrop<
    { form: FormGroup | FormField; index: number },
    unknown,
    { isOverCurrent: boolean; canDrop: boolean }
  >({
    accept: [FormKind.Group, FormKind.Field],
    canDrop(item, monitor) {
      const isOverCurrent = monitor.isOver({ shallow: true });
      return (
        isOverCurrent &&
        (item.form.kind !== FormKind.Group ||
          // cant dnd with deeper than 2 level group
          (level < 2 &&
            // cant dnd if sum of source children of group type (0 if none, 1 if children exist)
            // and target item's level is over 3 for field
            (item.form.children.filter((c) => c.kind === FormKind.Group).length === 0 ? 0 : 1) +
              level <
              3))
      );
    },
    collect: (monitor) => ({
      canDrop: monitor.canDrop(),
      isOverCurrent: monitor.isOver({ shallow: true }),
    }),
    hover(item) {
      const dragIndex = item.index;
      const hoverIndex = index;
      if (dragIndex !== hoverIndex && isOverCurrent && canDrop) {
        formStore.removeChild(item.form.id);
        formStore.addChild(parentId, item.form.kind, { index: hoverIndex, item: item.form });
        item.index = hoverIndex;
      }
    },
  });

  const captureEnterKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.stopPropagation();
        formStore.addChild(parentId, FormKind.Field, { index: index + 1, item: getInitField() });
      }
    },
    [formStore, index, parentId],
  );

  return (
    <div className={css.base} ref={(node) => drop(node)}>
      <ConjunctionContainer
        conjunction={conjunction}
        index={index}
        onClick={(value) => {
          formStore.setFieldConjunction(parentId, (value?.toString() ?? 'and') as Conjunction);
        }}
      />
      <div className={css.fieldCard} ref={preview}>
        <Select
          autoFocus
          dropdownMatchSelectWidth={250}
          options={columns.map((col) => ({
            label: col.displayName || col.column,
            value: col.column,
          }))}
          value={field.columnName}
          width={'100%'}
          onChange={onChangeColumnName}
        />
        <Select
          options={(isSpecialColumn
            ? [Operator.Eq, Operator.NotEq] // just Eq and NotEq for Special column
            : AvailableOperators[currentColumn?.type ?? V1ColumnType.UNSPECIFIED]
          ).map((op) => ({
            label: ReadableOperator[field.type][op],
            value: op,
          }))}
          value={field.operator}
          width={'100%'}
          onChange={(value) => {
            const op = (value?.toString() ?? '=') as Operator;
            formStore.setFieldOperator(field.id, op);
            if (op === Operator.IsEmpty || op === Operator.NotEmpty) {
              updateFieldValue(field.id, null);
            }
          }}
        />
        {isSpecialColumn ? (
          <div onKeyDownCapture={captureEnterKeyDown}>
            <Select
              options={getSpecialOptions(field.columnName as SpecialColumnNames)}
              value={fieldValue?.toString()}
              width={'100%'}
              onChange={(value) => {
                const val = value?.toString() ?? null;
                updateFieldValue(field.id, val);
              }}
            />
          </div>
        ) : (
          <>
            {(currentColumn?.type === V1ColumnType.TEXT ||
              currentColumn?.type === V1ColumnType.UNSPECIFIED) && (
              <Input
                disabled={
                  field.operator === Operator.IsEmpty || field.operator === Operator.NotEmpty
                }
                value={fieldValue?.toString() ?? undefined}
                onChange={(e) => {
                  const val = e.target.value || null; // when empty string, val is null
                  updateFieldValue(field.id, val, true);
                }}
                onPressEnter={() =>
                  formStore.addChild(parentId, FormKind.Field, {
                    index: index + 1,
                    item: getInitField(),
                  })
                }
              />
            )}
            {currentColumn?.type === V1ColumnType.NUMBER && (
              <InputNumber
                className={css.fullWidth}
                value={fieldValue != null ? Number(fieldValue) : undefined}
                onChange={(val) => {
                  const value = val != null ? Number(val) : null;
                  updateFieldValue(field.id, value, true);
                }}
                onPressEnter={() =>
                  formStore.addChild(parentId, FormKind.Field, {
                    index: index + 1,
                    item: getInitField(),
                  })
                }
              />
            )}
            {currentColumn?.type === V1ColumnType.DATE && (
              // dirty approach -- datePicker doesn't provide onPressEnter so
              // we override the behavior by attaching a handler higher up in
              // the tree in the capture phase
              <div onKeyDownCapture={captureEnterKeyDown}>
                {/* timezone is UTC since DB uses UTC */}
                <DatePicker
                  value={dayjs(fieldValue).isValid() ? dayjs(fieldValue).utc() : null}
                  onChange={(value: DatePickerProps['value']) => {
                    const dateString = dayjs(value).utc().startOf('date').format();
                    updateFieldValue(field.id, dateString);
                  }}
                />
              </div>
            )}
          </>
        )}
        <Button
          icon={<Icon name="close" title="close field" />}
          type="text"
          onClick={() => formStore.removeChild(field.id)}
        />
        <Button type="text">
          <div ref={drag}>
            <HolderOutlined />
          </div>
        </Button>
      </div>
    </div>
  );
};

export default FilterField;
