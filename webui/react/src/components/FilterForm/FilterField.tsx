import { DeleteOutlined, HolderOutlined } from '@ant-design/icons';
import { Select } from 'antd';
import { useDrag, useDrop } from 'react-dnd';

import Button from 'components/kit/Button';
import Input from 'components/kit/Input';
import InputNumber from 'components/kit/InputNumber';

import css from './FilterField.module.scss';
import { FormClassStore } from './FilterFormStore';
import {
  AvaliableOperators,
  ColumnType,
  Conjunction,
  FormField,
  FormGroup,
  FormType,
  Operator,
} from './type';

interface Props {
  index: number; // start from 0
  field: FormField;
  parentId: string;
  conjunction: Conjunction;
  formClassStore: FormClassStore;
  level: number; // start from 0
}

const FilterField = ({
  field,
  conjunction,
  formClassStore,
  index,
  parentId,
  level,
}: Props): JSX.Element => {
  const avaliableOperators = AvaliableOperators[ColumnType[field.columnName]];
  const [, drag, preview] = useDrag<{ form: FormField; index: number }, unknown, unknown>(() => ({
    item: { form: field, index },
    type: FormType.Field,
  }));

  const [{ isOverCurrent, canDrop }, drop] = useDrop<
    { form: FormGroup | FormField; index: number },
    unknown,
    { isOverCurrent: boolean; canDrop: boolean }
  >({
    accept: [FormType.Group, FormType.Field],
    canDrop(item, monitor) {
      const isOverCurrent = monitor.isOver({ shallow: true });
      if (isOverCurrent) {
        if (item.form.type === FormType.Group) {
          return (
            // cant dnd with deeper than 2 level group
            level < 2 &&
            // cant dnd if sum of source children of group type (0 if none, 1 if children exist)
            // and target item's level is over 3 for field
            (item.form.children.filter((c) => c.type === FormType.Group).length === 0 ? 0 : 1) +
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
        formClassStore.removeChild(item.form.id);
        formClassStore.addChild(parentId, item.form.type, hoverIndex, item.form);
        item.index = hoverIndex;
      }
    },
  });

  return (
    <div className={css.base} ref={(node) => drop(node)}>
      <>
        {index === 0 && <div>if</div>}
        {index === 1 && (
          <Select
            value={conjunction}
            onChange={(value: string) => {
              formClassStore.setFieldValue(parentId, 'conjunction', value);
            }}>
            {Object.values(Conjunction).map((c) => (
              <Select.Option key={c} value={c}>
                {c}
              </Select.Option>
            ))}
          </Select>
        )}
        {index > 1 && <div className={css.conjunction}>{conjunction}</div>}
      </>
      <div className={css.fieldCard} ref={preview}>
        <Select
          value={field.columnName}
          onChange={(value: string) => {
            const prevType = ColumnType[field.columnName];
            formClassStore.setFieldValue(field.id, 'columnName', value);
            if (prevType !== ColumnType[field.columnName]) {
              // change default operator every time columnName is changed
              formClassStore.setFieldValue(field.id, 'operator', avaliableOperators[0]);
              formClassStore.setFieldValue(field.id, 'value', null);
            }
          }}>
          <Select.Option value="id">id</Select.Option>
          <Select.Option value="tags">tags</Select.Option>
          <Select.Option value="state">state</Select.Option>
          <Select.Option value="user">user</Select.Option>
        </Select>
        <Select
          style={{ width: '100%' }}
          value={field.operator}
          onChange={(value: Operator) => {
            formClassStore.setFieldValue(field.id, 'operator', value);
          }}>
          {avaliableOperators.map((op) => (
            <Select.Option key={op} value={op}>
              {op}
            </Select.Option>
          ))}
        </Select>
        <>
          {ColumnType[field.columnName] === 'string' && (
            <Input
              value={field.value?.toString()}
              onChange={(e) => formClassStore.setFieldValue(field.id, 'value', e.target.value)}
            />
          )}
          {ColumnType[field.columnName] === 'number' && (
            <InputNumber
              value={Number(field.value ?? 0)}
              onChange={(val) => formClassStore.setFieldValue(field.id, 'value', Number(val))}
            />
          )}
        </>
        <Button
          icon={<DeleteOutlined />}
          type="text"
          onClick={() => formClassStore.removeChild(field.id)}
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
