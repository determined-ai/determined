import { DeleteOutlined, HolderOutlined, PlusOutlined } from '@ant-design/icons';
import { Dropdown, DropDownProps, Select } from 'antd';
import type { MenuProps } from 'antd';
import { useMemo } from 'react';
import { useDrag, useDrop } from 'react-dnd';

import Button from 'components/kit/Button';

import FilterField from './FilterField';
import { FormClassStore } from './FilterFormStore';
import css from './FilterGroup.module.scss';
import { Conjunction, FormField, FormGroup, FormType } from './type';

interface Props {
  conjunction: Conjunction;
  index: number; // start from 0
  group: FormGroup;
  parentId: string;
  level: number; // start from 0
  formClassStore: FormClassStore;
}

const FilterGroup = ({
  index,
  conjunction,
  group,
  level,
  formClassStore,
  parentId,
}: Props): JSX.Element => {
  const [, drag, preview] = useDrag<{ form: FormGroup; index: number }, unknown, unknown>(() => ({
    item: { form: group, index },
    type: FormType.Group,
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
            // cant self dnd
            group.id !== item.form.id &&
            // cant dnd in self childrens group
            item.form.children.filter((c) => c.id === group.id).length === 0 &&
            // cant dnd with deeper than 2 level group
            level < 2 &&
            // cant dnd if sum of source children of group type (0 if none, 1 if children exist)
            // and target item's level is over 2
            // 2 is the max depth
            (item.form.children.filter((c) => c.type === FormType.Group).length === 0 ? 0 : 1) +
              level <
              2
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
        formClassStore.addChild(group.id, item.form.type, hoverIndex, item.form);
        item.index = hoverIndex;
      }
    },
  });

  const menuItems: DropDownProps['menu'] = useMemo(() => {
    const onItemClick: MenuProps['onClick'] = (e) => {
      if (e.key === FormType.Field) {
        formClassStore.addChild(group.id, FormType.Field, group.children.length);
      } else if (e.key === FormType.Group) {
        formClassStore.addChild(group.id, FormType.Group, group.children.length);
      }
    };

    const items: MenuProps['items'] = [
      { icon: <PlusOutlined />, key: FormType.Field, label: <div>Add condition field</div> },
      {
        disabled: !(0 <= level && level <= 1),
        icon: <PlusOutlined />,
        key: FormType.Group,
        label: <div>Add condition group</div>,
      },
    ];
    return { items: items, onClick: onItemClick };
  }, [formClassStore, group.children.length, group.id, level]);

  return (
    <div className={`${css.base} ${level === 0 ? css.baseRoot : ''}`} ref={(node) => drop(node)}>
      {level > 0 && (
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
      )}
      <div className={css.groupCard} ref={preview}>
        <div className={css.header}>
          <div className={css.headerCaption}>
            {group.conjunction === Conjunction.And ? (
              <div>All of the following coditions are true</div>
            ) : (
              <div>Some of the following coditions are true</div>
            )}
          </div>
          <div className={css.headerButtonGroup}>
            <Dropdown menu={menuItems} trigger={['click']}>
              <Button icon={<PlusOutlined />} type="text" />
            </Dropdown>
            <Button
              icon={<DeleteOutlined />}
              type="text"
              onClick={() => formClassStore.removeChild(group.id)}
            />
            {level > 0 && (
              <div ref={drag}>
                <Button icon={<HolderOutlined />} type="text" />
              </div>
            )}
          </div>
        </div>
        <div className={css.children}>
          {group.children.map((child, i) => {
            if (child.type === FormType.Group) {
              return (
                <FilterGroup
                  conjunction={group.conjunction}
                  formClassStore={formClassStore}
                  group={child}
                  index={i}
                  key={child.id}
                  level={level + 1}
                  parentId={group.id}
                />
              );
            } else {
              return (
                <FilterField
                  conjunction={group.conjunction}
                  field={child}
                  formClassStore={formClassStore}
                  index={i}
                  key={child.id}
                  level={level + 1}
                  parentId={group.id}
                />
              );
            }
          })}
        </div>
        {level === 0 && (
          <div>
            <Button
              onClick={() =>
                formClassStore.addChild(group.id, FormType.Field, group.children.length)
              }>
              + Add condition field
            </Button>
            <Button
              onClick={() =>
                formClassStore.addChild(group.id, FormType.Group, group.children.length)
              }>
              + Add condition group
            </Button>
          </div>
        )}
      </div>
    </div>
  );
};

export default FilterGroup;
