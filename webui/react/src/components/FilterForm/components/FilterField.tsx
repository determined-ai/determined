import { DeleteOutlined, HolderOutlined } from '@ant-design/icons';
import { DatePicker } from 'antd';
import type { DatePickerProps } from 'antd/es/date-picker';
import dayjs from 'dayjs';
import { useDrag, useDrop } from 'react-dnd';

import Button from 'components/kit/Button';
import Input from 'components/kit/Input';
import InputNumber from 'components/kit/InputNumber';
import Select, { Option } from 'components/kit/Select';
import { V1ColumnType, V1ProjectColumn } from 'services/api-ts-sdk';

import ConjunctionContainer from './ConjunctionContainer';
import css from './FilterField.module.scss';
import { FilterFormStore } from './FilterFormStore';
import { AvaliableOperators, Conjunction, FormField, FormGroup, FormKind, Operator } from './type';

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
  const currentColumn = columns.find((c) => c.column === field.columnName);
  const [, drag, preview] = useDrag<{ form: FormField; index: number }, unknown, unknown>(() => ({
    item: { form: field, index },
    type: FormKind.Field,
  }));

  const [{ isOverCurrent, canDrop }, drop] = useDrop<
    { form: FormGroup | FormField; index: number },
    unknown,
    { isOverCurrent: boolean; canDrop: boolean }
  >({
    accept: [FormKind.Group, FormKind.Field],
    canDrop(item, monitor) {
      const isOverCurrent = monitor.isOver({ shallow: true });
      if (isOverCurrent) {
        if (item.form.kind === FormKind.Group) {
          return (
            // cant dnd with deeper than 2 level group
            level < 2 &&
            // cant dnd if sum of source children of group type (0 if none, 1 if children exist)
            // and target item's level is over 3 for field
            (item.form.children.filter((c) => c.kind === FormKind.Group).length === 0 ? 0 : 1) +
              level <
              3
          );
        }
        return true;
      }
      return false;
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
        formStore.addChild(parentId, item.form.kind, hoverIndex, item.form);
        item.index = hoverIndex;
      }
    },
  });

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
          dropdownMatchSelectWidth={250} // TODO(fix): set corrent width
          searchable={false}
          value={field.columnName}
          width={'100%'}
          onChange={(value) => {
            const prevType = currentColumn?.type;
            const newCol = columns.find((c) => c.column === value?.toString() ?? '');
            if (newCol) {
              formStore.setFieldColumnName(field.id, newCol);
            }
            if (prevType !== newCol?.type) {
              const defaultOperator: Operator =
                AvaliableOperators[newCol?.type ?? V1ColumnType.UNSPECIFIED][0];
              formStore.setFieldOperator(field.id, defaultOperator);
              formStore.setFieldValue(field.id, null);
            }
          }}>
          {columns.map((col) => (
            <Option key={col.column} value={col.column}>
              {col.displayName || col.column}
            </Option>
          ))}
        </Select>
        <Select
          searchable={false}
          value={field.operator}
          width={'100%'}
          onChange={(value) => {
            formStore.setFieldOperator(field.id, (value?.toString() ?? '=') as Operator);
          }}>
          {AvaliableOperators[currentColumn?.type ?? V1ColumnType.UNSPECIFIED].map((op) => (
            <Option key={op} value={op}>
              {op}
            </Option>
          ))}
        </Select>
        <>
          {(currentColumn?.type === V1ColumnType.TEXT ||
            currentColumn?.type === V1ColumnType.UNSPECIFIED) && (
            <Input
              disabled={field.operator === Operator.isEmpty || field.operator === Operator.notEmpty}
              value={
                field.operator === Operator.isEmpty || field.operator === Operator.notEmpty
                  ? undefined
                  : field.value?.toString()
              }
              onChange={(e) => formStore.setFieldValue(field.id, e.target.value)}
            />
          )}
          {currentColumn?.type === V1ColumnType.NUMBER && (
            <InputNumber
              className={css.fullWidth}
              value={field.value != null ? Number(field.value) : undefined}
              onChange={(val) => {
                formStore.setFieldValue(field.id, val != null ? Number(val) : null);
              }}
            />
          )}
          {currentColumn?.type === V1ColumnType.DATE && (
            // timezone is UTC since DB uses UTC
            <DatePicker
              value={dayjs(field.value).isValid() ? dayjs(field.value).utc() : null}
              onChange={(value: DatePickerProps['value']) => {
                const dateString = dayjs(value).utc().startOf('date').format();
                formStore.setFieldValue(field.id, dateString);
              }}
            />
          )}
        </>
        <Button
          icon={<DeleteOutlined />}
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
