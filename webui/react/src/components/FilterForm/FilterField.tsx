import { DeleteOutlined, HolderOutlined } from '@ant-design/icons';
import { useDrag, useDrop } from 'react-dnd';

import Button from 'components/kit/Button';
import Input from 'components/kit/Input';
import InputNumber from 'components/kit/InputNumber';
import Select, { Option } from 'components/kit/Select';

import ConjunctionContainer from './ConjunctionContainer';
import css from './FilterField.module.scss';
import { FilterFormStore } from './FilterFormStore';
import {
  AvaliableOperators,
  ColumnType,
  Conjunction,
  FormField,
  FormGroup,
  FormKind,
  Operator,
} from './type';

interface Props {
  index: number; // start from 0
  field: FormField;
  parentId: string;
  conjunction: Conjunction;
  formStore: FilterFormStore;
  level: number; // start from 0
}

const FilterField = ({
  field,
  conjunction,
  formStore,
  index,
  parentId,
  level,
}: Props): JSX.Element => {
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
          formStore.setFieldValue(parentId, 'conjunction', value?.toString() ?? '');
        }}
      />
      <div className={css.fieldCard} ref={preview}>
        <Select
          dropdownMatchSelectWidth={250} // TODO(fix): set corrent width
          searchable={false}
          value={field.columnName}
          width={'100%'}
          onChange={(value) => {
            const prevType = ColumnType[field.columnName];
            formStore.setFieldValue(field.id, 'columnName', value?.toString() ?? '');
            if (prevType !== ColumnType[field.columnName]) {
              const defaultOperator = AvaliableOperators[ColumnType[field.columnName]][0];
              formStore.setFieldValue(field.id, 'operator', defaultOperator);
              formStore.setFieldValue(field.id, 'value', null);
            }
          }}>
          {Object.keys(ColumnType).map((col) => (
            <Option key={col} value={col}>
              {col}
            </Option>
          ))}
        </Select>
        <Select
          searchable={false}
          value={field.operator}
          width={'100%'}
          onChange={(value) => {
            formStore.setFieldValue(field.id, 'operator', value?.toString() ?? '');
          }}>
          {AvaliableOperators[ColumnType[field.columnName]].map((op) => (
            <Option key={op} value={op}>
              {op}
            </Option>
          ))}
        </Select>
        <>
          {ColumnType[field.columnName] === 'string' && (
            <Input
              disabled={field.operator === Operator.isEmpty || field.operator === Operator.notEmpty}
              value={
                field.operator === Operator.isEmpty || field.operator === Operator.notEmpty
                  ? undefined
                  : field.value?.toString()
              }
              onChange={(e) => formStore.setFieldValue(field.id, 'value', e.target.value)}
            />
          )}
          {ColumnType[field.columnName] === 'number' && (
            <InputNumber
              className={css.fullWidth}
              value={field.value != null ? Number(field.value) : undefined}
              onChange={(val) => {
                formStore.setFieldValue(field.id, 'value', val != null ? Number(val) : null);
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
