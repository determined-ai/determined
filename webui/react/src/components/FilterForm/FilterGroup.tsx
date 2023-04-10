import { DeleteOutlined, HolderOutlined, PlusOutlined } from '@ant-design/icons';
import { Dropdown, DropDownProps } from 'antd';
import type { MenuProps } from 'antd';
import { useMemo } from 'react';
import { useDrag, useDrop } from 'react-dnd';

import Button from 'components/kit/Button';

import ConjunctionContainer from './ConjunctionContainer';
import FilterField from './FilterField';
import { FilterFormStore } from './FilterFormStore';
import css from './FilterGroup.module.scss';
import { Conjunction, FormField, FormGroup, FormType } from './type';

interface Props {
  conjunction: Conjunction;
  index: number; // start from 0
  group: FormGroup;
  parentId: string;
  level: number; // start from 0
  formStore: FilterFormStore;
}

const FilterGroup = ({
  index,
  conjunction,
  group,
  level,
  formStore,
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
    hover(item, monitor) {
      const dragIndex = item.index;
      const hoverIndex = index;
      const diffOffset = monitor.getDifferenceFromInitialOffset();

      if (isOverCurrent && canDrop) {
        formStore.removeChild(item.form.id);
        const insertIndex = (() => {
          if (dragIndex !== hoverIndex) {
            return hoverIndex;
          } else {
            // if drag on level=0 group, insert on the top if diffOffset is lower than original position
            return (diffOffset?.y ?? 0) > 0 ? group.children.length : 0;
          }
        })();
        formStore.addChild(group.id, item.form.type, insertIndex, item.form);
        item.index = hoverIndex;
      }
    },
  });

  const menuItems: DropDownProps['menu'] = useMemo(() => {
    const onItemClick: MenuProps['onClick'] = (e) => {
      if (e.key === FormType.Field) {
        formStore.addChild(group.id, FormType.Field, group.children.length);
      } else if (e.key === FormType.Group) {
        formStore.addChild(group.id, FormType.Group, group.children.length);
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
  }, [formStore, group.children.length, group.id, level]);

  if (level === 0 && group.children.length === 0) {
    // return empty div if there's nothing to show
    return <div />;
  }

  return (
    <div className={`${css.base} ${level === 0 ? css.baseRoot : ''}`} ref={(node) => drop(node)}>
      {level > 0 && (
        <ConjunctionContainer
          conjunction={conjunction}
          index={index}
          onClick={(value) => {
            formStore.setFieldValue(parentId, 'conjunction', value?.toString() ?? '');
          }}
        />
      )}
      <div className={`${css.groupCard} ${css[`level${level}`]}`} ref={preview}>
        <div className={css.header}>
          <div>
            {group.conjunction === Conjunction.And ? (
              <div>All of the following conditions are true</div>
            ) : (
              <div>Some of the following conditions are true</div>
            )}
          </div>
          {level > 0 && (
            <div className={css.headerButtonGroup}>
              <Dropdown menu={menuItems} trigger={['click']}>
                <Button icon={<PlusOutlined />} type="text" />
              </Dropdown>
              <Button
                icon={<DeleteOutlined />}
                type="text"
                onClick={() => formStore.removeChild(group.id)}
              />
              <div ref={drag}>
                <Button icon={<HolderOutlined />} type="text" />
              </div>
            </div>
          )}
        </div>
        <div className={css.children}>
          {group.children.map((child, i) => {
            if (child.type === FormType.Group) {
              return (
                <FilterGroup
                  conjunction={group.conjunction}
                  formStore={formStore}
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
                  formStore={formStore}
                  index={i}
                  key={child.id}
                  level={level + 1}
                  parentId={group.id}
                />
              );
            }
          })}
        </div>
      </div>
    </div>
  );
};

export default FilterGroup;
