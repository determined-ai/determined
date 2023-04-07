import { DeleteOutlined, HolderOutlined, PlusOutlined } from '@ant-design/icons';
import { Dropdown, DropDownProps, Select } from 'antd';
import type { MenuProps } from 'antd';
import { useMemo } from 'react';
import { useDrag, useDrop } from 'react-dnd';

import Button from 'components/kit/Button';
import { ValueOf } from 'shared/types';

import FilterField from './FilterField';
import { FormClassStore } from './FilterFormStore';
import css from './FilterGroup.module.scss';
import { Conjunction, FormField, FormGroup, ItemTypes } from './type';

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
    item: () => {
      return { form: group, index };
    },
    type: ItemTypes.GROUP,
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
            // cant self dnd
            group.id !== item.form.id &&
            // cant dnd in self childrens group
            item.form.children.filter((c) => c.id === group.id).length === 0 &&
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
        formClassStore.addChild(group.id, item.form.type, hoverIndex, item.form);
        item.index = hoverIndex;
      }
    },
  });

  const menuItems: DropDownProps['menu'] = useMemo(() => {
    const MenuKey = {
      Field: 'field',
      Group: 'group',
    } as const;

    const funcs = {
      [MenuKey.Field]: () => {
        formClassStore.addChild(group.id, 'field', group.children.length);
      },
      [MenuKey.Group]: () => {
        formClassStore.addChild(group.id, 'group', group.children.length);
      },
    };

    const onItemClick: MenuProps['onClick'] = (e) => {
      funcs[e.key as ValueOf<typeof MenuKey>]();
    };

    const items: MenuProps['items'] = [
      {
        icon: <PlusOutlined />,
        key: 'field',
        label: <div>Add condition field</div>,
      },
      {
        disabled: !(0 <= level && level <= 1),
        icon: <PlusOutlined />,
        key: 'group',
        label: <div>Add condition group</div>,
      },
    ];
    return { items: items, onClick: onItemClick };
  }, [formClassStore, group.children.length, group.id, level]);

  return (
    <div className={`${css.base} ${level === 0 ? css.baseRoot : ''}`} ref={(node) => drop(node)}>
      {level > 0 && (
        <>
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
        </>
      )}
      <div className={css.groupCard} ref={preview}>
        <div className={css.header}>
          <div className={css.headerCaption}>
            {group.conjunction === 'and' ? (
              <div>All of the following coditions are true</div>
            ) : (
              <div>Some of the following coditions are true</div>
            )}
          </div>
          <div className={css.headerButtonGroup}>
            <Button type="text">
              <Dropdown menu={menuItems} trigger={['click']}>
                <PlusOutlined />
              </Dropdown>
            </Button>
            <Button type="text" onClick={() => formClassStore.removeChild(group.id)}>
              {/* not using `icon` prop on purpose to get the same button layout as dropdown */}
              <DeleteOutlined />
            </Button>
            {level > 0 && (
              <div className={css.draggableHandle} ref={drag}>
                <Button type="text">
                  <HolderOutlined />
                </Button>
              </div>
            )}
          </div>
        </div>
        <div className={css.children}>
          {group.children.map((child, i) => {
            if (child.type === 'group') {
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
        <div className={css.footer}>
          {level === 0 && (
            <>
              <Button
                onClick={() => formClassStore.addChild(group.id, 'field', group.children.length)}>
                + Add condition field
              </Button>
              <Button
                onClick={() => formClassStore.addChild(group.id, 'group', group.children.length)}>
                + Add condition group
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default FilterGroup;
