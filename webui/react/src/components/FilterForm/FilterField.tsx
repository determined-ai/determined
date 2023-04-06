import { DeleteOutlined, HolderOutlined } from '@ant-design/icons';
import { Select } from 'antd';
import { useDrag } from 'react-dnd';

import Button from 'components/kit/Button';
import Input from 'components/kit/Input';
import InputNumber from 'components/kit/InputNumber';

import css from './FilterField.module.scss';
import { FormClassStore } from './FilterFormStore';
import { Conjunction, FormField, ItemTypes, Operator, OperatorMap } from './type';

interface Props {
  index: number; // start from 0
  field: FormField;
  parentId: string;
  conjunction: Conjunction;
  formClassStore: FormClassStore;
}

const FilterField = ({
  field,
  conjunction,
  formClassStore,
  index,
  parentId,
}: Props): JSX.Element => {
  const [, drag, preview] = useDrag<FormField, unknown, unknown>(() => ({
    item: field,
    type: ItemTypes.FIELD,
  }));

  return (
    <div className={css.base}>
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
