import { DeleteOutlined, HolderOutlined } from '@ant-design/icons';
import { Select } from 'antd';
import { useDrag, useDrop } from 'react-dnd';

import Button from 'components/kit/Button';
import Input from 'components/kit/Input';
import InputNumber from 'components/kit/InputNumber';

import css from './FilterField.module.scss';
import { FormClassStore } from './FilterFormStore';
import { Conjunction, FormField, FormGroup, ItemTypes, Operator, OperatorMap } from './type';

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
  const [, drag, preview] = useDrag<{ form: FormField; index: number }, unknown, unknown>(() => ({
    item: () => {
      return { form: field, index };
    },
    type: ItemTypes.FIELD,
  }));

  const [{ isOverCurrent, canDrop }, drop] = useDrop<
    { form: FormGroup | FormField; index: number },
    unknown,
    { isOverCurrent: boolean; canDrop: boolean }
  >({
    accept: [ItemTypes.GROUP, ItemTypes.FIELD],
    canDrop(item, monitor) {
      const isOverCurrent = monitor.isOver({ shallow: true });
      if (isOverCurrent) {
        if (item.form.type === 'group') {
          return (
            // cant dnd with deeper than 2 level group
            level < 2 &&
            // cant dnd if sum of source children of group type (0 if none, 1 if children exist)
            // and target item's level is over 2
            // 2 is the max depth
            (item.form.children.filter((c) => c.type === 'group').length === 0 ? 0 : 1) + level < 2
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
      if (dragIndex === hoverIndex) {
        return;
      }
      if (isOverCurrent && canDrop) {
        formClassStore.removeChild(item.form.id);
        formClassStore.addChild(parentId, item.form.type, hoverIndex, item.form);
        item.index = hoverIndex;
      }
    },
  });

  return (
    <div className={css.base} ref={(node) => drop(node)}>
      {index === 0 ? (
        <div>where</div>
      ) : (
        <>
          {index === 1 ? (
            <Select
              value={conjunction}
              onChange={(value: string) => {
                formClassStore.setFieldValue(parentId, 'conjunction', value);
              }}>
              <Select.Option value="and">and</Select.Option>
              <Select.Option value="or">or</Select.Option>
            </Select>
          ) : (
            <div className={css.conjunction}>{conjunction}</div>
          )}
        </>
      )}
      <div className={css.fieldCard} ref={preview}>
        <Select
          value={field.columnName}
          onChange={(value: string) => {
            formClassStore.setFieldValue(field.id, 'columnName', value);
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
          {Object.entries(OperatorMap).map((op) => (
            <Select.Option key={op[0]} value={op[0]}>
              {op[1]}
            </Select.Option>
          ))}
        </Select>
        {['string'].includes(field.columnName) ? (
          <Input size="small" value={field.value?.toString()} />
        ) : (
          <InputNumber value={field.value as number} />
        )}
        <Button
          icon={<DeleteOutlined />}
          type="text"
          onClick={() => formClassStore.removeChild(field.id)}
        />
        <div className={css.draggableHandle} ref={drag}>
          <Button type="text">
            <HolderOutlined />
          </Button>
        </div>
      </div>
    </div>
  );
};

export default FilterField;
