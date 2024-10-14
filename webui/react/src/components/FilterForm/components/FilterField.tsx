import dayjs from 'dayjs';
import Badge from 'hew/Badge';
import Button from 'hew/Button';
import DatePicker, { DatePickerProps } from 'hew/DatePicker';
import Icon from 'hew/Icon';
import Input from 'hew/Input';
import InputNumber from 'hew/InputNumber';
import InputSelect from 'hew/InputSelect';
import Select, { SelectProps, SelectValue } from 'hew/Select';
import { Loadable, NotLoaded } from 'hew/utils/loadable';
import { Observable, useObservable } from 'micro-observables';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useDrag, useDrop } from 'react-dnd';
import { debounce } from 'throttle-debounce';

import ConjunctionContainer from 'components/FilterForm/components/ConjunctionContainer';
import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
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
import { useAsync } from 'hooks/useAsync';
import { getMetadataValues } from 'services/api';
import { V1ColumnType, V1LocationType, V1ProjectColumn } from 'services/api-ts-sdk';
import clusterStore from 'stores/cluster';
import userStore from 'stores/users';
import { alphaNumericSorter } from 'utils/sort';

import css from './FilterField.module.scss';

const debounceFunc = debounce(1000, (func: () => void) => {
  func();
});

const COLUMN_TYPE = {
  NormalColumnType: 'NormalColumnType',
  SpecialColumnType: 'SpecialColumnType',
  StringMetadataColumnType: 'StringMetadataColumn',
} as const;

interface Props {
  index: number; // start from 0
  field: FormField;
  parentId: string;
  conjunction: Conjunction;
  formStore: FilterFormStore;
  level: number; // start from 0
  columns: V1ProjectColumn[];
  projectId?: number;
}

const FilterField = ({
  field,
  conjunction,
  formStore,
  index,
  parentId,
  level,
  columns,
  projectId,
}: Props): JSX.Element => {
  const users = Loadable.getOrElse([], useObservable(userStore.getUsers()));
  const resourcePools = Loadable.getOrElse([], useObservable(clusterStore.resourcePools));
  const currentColumn = useMemo(
    () => columns.find((c) => c.column === field.columnName),
    [columns, field.columnName],
  );
  const [metadataColumns, setMetadataColumns] = useState(() => new Map<string, number[]>()); // a map of metadata columns and found indexes

  useEffect(() => {
    for (const [index, { column }] of columns.entries()) {
      if (column.includes('metadata')) {
        const columnEntry = metadataColumns.get(column) ?? [];
        if (!columnEntry.includes(index)) {
          setMetadataColumns((prev) => {
            prev.set(column, [...columnEntry, index]);
            return prev;
          });
        }
      }
    }
  }, [columns, metadataColumns]);

  const columnType = useMemo(() => {
    if (field.location === V1LocationType.RUNMETADATA && field.type === V1ColumnType.TEXT) {
      return COLUMN_TYPE.StringMetadataColumnType;
    } else if ((SpecialColumnNames as ReadonlyArray<string>).includes(field.columnName)) {
      return COLUMN_TYPE.SpecialColumnType;
    }
    return COLUMN_TYPE.NormalColumnType;
  }, [field.columnName, field.location, field.type]);

  const [inputOpen, setInputOpen] = useState(false);
  const [fieldValue, setFieldValue] = useState<FormFieldValue>(field.value);

  // use this function to update field value
  const updateFieldValue = (fieldId: string, value: FormFieldValue, debounceUpdate = false) => {
    Observable.batch(() => {
      if (debounceUpdate) {
        debounceFunc(() => formStore.setFieldValue(fieldId, value));
      } else {
        formStore.setFieldValue(fieldId, value);
      }
      setFieldValue(value);
    });
  };

  const onChangeColumnName = (value: SelectValue) => {
    const prevType = field.type;
    const [newColName, type] = (value?.toString() ?? '').split(' ');
    const newCol = columns.find((c) => c.column === newColName && type === c.type);
    if (newCol) {
      Observable.batch(() => {
        formStore.setFieldColumnName(field.id, newCol);
        if ((SpecialColumnNames as ReadonlyArray<string>).includes(newColName)) {
          formStore.setFieldOperator(field.id, Operator.Eq);
          updateFieldValue(field.id, null);
        } else if (prevType !== newCol.type) {
          const defaultOperator: Operator =
            AvailableOperators[newCol.type ?? V1ColumnType.UNSPECIFIED][0];
          formStore.setFieldOperator(field.id, defaultOperator);
          updateFieldValue(field.id, null);
        }
      });
    }
  };

  const metadataValues = useAsync(async () => {
    try {
      if (projectId !== undefined && columnType === COLUMN_TYPE.StringMetadataColumnType) {
        const metadataValues = await getMetadataValues({
          key: field.columnName.replace(/^metadata\./, ''),
          projectId,
        });
        return metadataValues;
      }
      return [];
    } catch (e) {
      return NotLoaded;
    }
  }, [columnType, field.columnName, projectId]);

  const getSpecialOptions = (columnName: SpecialColumnNames): SelectProps['options'] => {
    switch (columnName) {
      case 'resourcePool':
        return resourcePools.map((rp) => ({ label: rp.name, value: rp.name }));
      case 'state':
        return RUN_STATES.map((state) => ({ label: state, value: state }));
      case 'searcherType':
        return SEARCHER_TYPE.map((searcher) => ({ label: searcher, value: searcher }));
      case 'user':
        return (
          users
            .map((user) => ({
              label: user.displayName || user.username,
              value: user.id.toString(),
            }))
            // getUsers sorts the users similarly but uses nullish coalescing
            // which doesn't work because the backend sends null strings in the
            // database as empty strings
            .sort((a, b) => alphaNumericSorter(a.label, b.label))
        );
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
      // would use isComposing alone but safari has a bug: https://bugs.webkit.org/show_bug.cgi?id=165004
      if (e.key === 'Enter' && !inputOpen && !e.nativeEvent.isComposing && e.keyCode !== 229) {
        formStore.addChild(parentId, FormKind.Field, {
          index: index + 1,
          item: formStore.newField(),
        });
        // stop panel flashing for selects and dates
        if (
          field.type === 'COLUMN_TYPE_DATE' ||
          columnType === COLUMN_TYPE.SpecialColumnType ||
          columnType === COLUMN_TYPE.StringMetadataColumnType
        ) {
          e.stopPropagation();
        }
      }
    },
    [columnType, field.type, formStore, index, inputOpen, parentId],
  );

  const getColDisplayName = (col: V1ProjectColumn) => {
    const metCol = metadataColumns.get(col.column);
    if (metCol !== undefined && metCol.length > 1) {
      return (
        <>
          {col.column} <Badge text={col.type.replace('COLUMN_TYPE_', '').toLowerCase()} />
        </>
      );
    }

    return col.displayName || col.column;
  };

  return (
    <div className={css.base} data-test-component="FilterField" ref={(node) => drop(node)}>
      <ConjunctionContainer
        conjunction={conjunction}
        index={index}
        onClick={(value) => {
          formStore.setFieldConjunction(parentId, (value?.toString() ?? 'and') as Conjunction);
        }}
      />
      <div className={css.fieldCard} data-test="fieldCard" ref={preview}>
        <Select
          autoFocus
          data-test="columnName"
          dropdownMatchSelectWidth={300}
          options={columns.map((col, idx) => ({
            key: `${col.column} ${idx}`,
            label: getColDisplayName(col),
            value: `${col.column} ${col.type}`,
          }))}
          value={`${field.columnName} ${field.type}`}
          width={'100%'}
          onChange={onChangeColumnName}
        />
        <Select
          data-test="operator"
          options={{
            [COLUMN_TYPE.StringMetadataColumnType]: [
              Operator.Contains,
              Operator.Eq,
              Operator.NotEq,
            ],
            [COLUMN_TYPE.SpecialColumnType]: [Operator.Eq, Operator.NotEq],
            [COLUMN_TYPE.NormalColumnType]:
              AvailableOperators[currentColumn?.type ?? V1ColumnType.UNSPECIFIED],
          }[columnType].map((op) => ({
            label: ReadableOperator[field.type][op],
            value: op,
          }))}
          value={field.operator}
          width={'100%'}
          onChange={(value) => {
            Observable.batch(() => {
              const op = (value?.toString() ?? '=') as Operator;
              formStore.setFieldOperator(field.id, op);
              if (op === Operator.IsEmpty || op === Operator.NotEmpty) {
                updateFieldValue(field.id, null);
              }
            });
          }}
        />
        {columnType !== COLUMN_TYPE.NormalColumnType ? (
          columnType === COLUMN_TYPE.StringMetadataColumnType ? (
            // StringMetadataColumnType
            <InputSelect
              customFilter={(options, filterValue) =>
                options.filter((opt) => opt.includes(filterValue))
              }
              data-test="special"
              options={metadataValues.getOrElse([])}
              value={typeof fieldValue === 'string' ? fieldValue : undefined}
              width={'100%'}
              onChange={(value) => {
                updateFieldValue(field.id, value);
              }}
              onDropdownVisibleChange={setInputOpen}
            />
          ) : (
            // SpecialColumnType
            <div onKeyDownCapture={captureEnterKeyDown}>
              <Select
                data-test="special"
                options={getSpecialOptions(field.columnName as SpecialColumnNames)}
                value={fieldValue ?? undefined}
                width={'100%'}
                onChange={(value) => {
                  const val = value?.toString() ?? null;
                  updateFieldValue(field.id, val);
                }}
                onDropdownVisibleChange={setInputOpen}
              />
            </div>
          )
        ) : (
          <>
            {(currentColumn?.type === V1ColumnType.TEXT ||
              currentColumn?.type === V1ColumnType.UNSPECIFIED) && (
              <Input
                data-test="text"
                disabled={
                  field.operator === Operator.IsEmpty || field.operator === Operator.NotEmpty
                }
                value={fieldValue?.toString() ?? undefined}
                onChange={(e) => {
                  const val = e.target.value || null; // when empty string, val is null
                  updateFieldValue(field.id, val, true);
                }}
                onPressEnter={captureEnterKeyDown}
              />
            )}
            {currentColumn?.type === V1ColumnType.NUMBER && (
              <InputNumber
                className={css.fullWidth}
                data-test="number"
                value={fieldValue != null ? Number(fieldValue) : undefined}
                onChange={(val) => {
                  const value = val != null ? Number(val) : null;
                  updateFieldValue(field.id, value, true);
                }}
                onPressEnter={captureEnterKeyDown}
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
                    const dt = dayjs(value).utc().startOf('date');
                    updateFieldValue(field.id, dt.isValid() ? dt.format() : null);
                  }}
                  onOpenChange={setInputOpen}
                />
              </div>
            )}
          </>
        )}
        <Button
          data-test="remove"
          icon={<Icon name="close" size="tiny" title="Close Field" />}
          type="text"
          onClick={() => formStore.removeChild(field.id)}
        />
        <Button data-test="move" type="text">
          <div ref={drag}>
            <Icon name="holder" size="small" title="Move field" />
          </div>
        </Button>
      </div>
    </div>
  );
};

export default FilterField;
