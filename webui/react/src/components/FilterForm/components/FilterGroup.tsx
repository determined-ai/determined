import { HolderOutlined, PlusOutlined } from '@ant-design/icons';
import { Dropdown, DropDownProps, type MenuProps } from 'antd';
import { useMemo, useRef } from 'react';
import { useDrag, useDrop } from 'react-dnd';

import ConjunctionContainer from 'components/FilterForm/components/ConjunctionContainer';
import FilterField from 'components/FilterForm/components/FilterField';
import { FilterFormStore, ITEM_LIMIT } from 'components/FilterForm/components/FilterFormStore';
import css from 'components/FilterForm/components/FilterGroup.module.scss';
import { Conjunction, FormField, FormGroup, FormKind } from 'components/FilterForm/components/type';
import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import { V1ProjectColumn } from 'services/api-ts-sdk';

interface Props {
  conjunction: Conjunction;
  index: number; // start from 0
  group: FormGroup;
  parentId: string;
  level: number; // start from 0
  formStore: FilterFormStore;
  columns: V1ProjectColumn[];
}

const FilterGroup = ({
  index,
  conjunction,
  group,
  level,
  formStore,
  parentId,
  columns,
}: Props): JSX.Element => {
  const scrollBottomRef = useRef<HTMLDivElement>(null);
  const [, drag, preview] = useDrag<{ form: FormGroup; index: number }, unknown, unknown>(
    () => ({
      item: { form: group, index },
      type: FormKind.Group,
    }),
    [group],
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
          // cant self dnd
          (group.id !== item.form.id &&
            // cant dnd in self childrens group
            item.form.children.filter((c) => c.id === group.id).length === 0 &&
            // cant dnd with deeper than 2 level group
            level < 2 &&
            // cant dnd if sum of source children of group type (0 if none, 1 if children exist)
            // and target item's level is over 2
            // 2 is the max depth
            (item.form.children.filter((c) => c.kind === FormKind.Group).length === 0 ? 0 : 1) +
              level <
              2))
      );
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
        formStore.addChild(group.id, item.form.kind, { index: insertIndex, item: item.form });
        item.index = hoverIndex;
      }
    },
  });

  const menuItems: DropDownProps['menu'] = useMemo(() => {
    const onItemClick: MenuProps['onClick'] = (e) => {
      if (e.key === FormKind.Field || e.key === FormKind.Group) {
        formStore.addChild(group.id, e.key);
        setTimeout(() => {
          scrollBottomRef?.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
        }, 100);
      }
    };

    const items: MenuProps['items'] = [
      { key: FormKind.Field, label: <div>Add condition</div> },
      {
        disabled: level > 1,
        key: FormKind.Group,
        label: <div>Add condition group</div>,
      },
    ];
    return { items: items, onClick: onItemClick };
  }, [formStore, group.id, level]);

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
            formStore.setFieldConjunction(parentId, (value?.toString() ?? 'and') as Conjunction);
          }}
        />
      )}
      <div className={`${css.groupCard} ${css[`level${level}`]}`} ref={preview}>
        <div className={css.header}>
          {level > 0 && (
            <>
              <div>
                {group.conjunction === Conjunction.And ? (
                  <div>All of the following are true...</div>
                ) : (
                  <div>Any of the following are true...</div>
                )}
              </div>
              <Dropdown
                disabled={group.children.length > ITEM_LIMIT}
                menu={menuItems}
                trigger={['click']}>
                <Button icon={<PlusOutlined />} type="text" />
              </Dropdown>
              <Button
                icon={<Icon name="close" title="Close Group" />}
                type="text"
                onClick={() => formStore.removeChild(group.id)}
              />
              <div ref={drag}>
                <Button icon={<HolderOutlined />} type="text" />
              </div>
            </>
          )}
        </div>
        <div className={css.children}>
          {group.children.map((child, i) => {
            if (child.kind === FormKind.Group) {
              return (
                <FilterGroup
                  columns={columns}
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
                  columns={columns}
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
          <div ref={scrollBottomRef} />
        </div>
      </div>
    </div>
  );
};

export default FilterGroup;
