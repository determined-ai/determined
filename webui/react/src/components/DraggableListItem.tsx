import React, { CSSProperties } from 'react';
import { useDrag, useDrop } from 'react-dnd';

import css from './DraggableListItem.module.scss';

interface Props {
  children: React.ReactNode;
  columnName: string;
  index: number;
  onClick: (event: React.MouseEvent) => void;
  onDrop: (column: string, newNeighborColumnName: string) => void;
  style?: CSSProperties;
}
interface DroppableItemProps {
  columnName: string;
  index: number;
}

const DraggableTypes = { COLUMN: 'COLUMN' };

const DraggableListItem: React.FC<Props> = ({
  columnName,
  index,
  style,
  onClick,
  children,
  onDrop,
}: Props) => {
  const [{ isOver, dropDirection }, drop] = useDrop(
    () => ({
      accept: DraggableTypes.COLUMN,
      collect: (monitor) => ({
        canDrop: monitor.canDrop(),
        dropDirection: monitor.getItem()?.index > index ? 'above' : 'below',
        isOver: monitor.isOver(),
      }),
      drop: (item: DroppableItemProps) => {
        const dragIndex = item.index;
        const targetIndex = index;

        // source and target item should be different items
        if (dragIndex !== targetIndex) {
          onDrop(item.columnName, columnName);
        }
      },
    }),
    [columnName, index],
  );

  const [, drag] = useDrag(
    () => ({
      item: { columnName, index },
      type: DraggableTypes.COLUMN,
    }),
    [columnName, index],
  );

  return (
    <li
      className={
        isOver
          ? dropDirection === 'above'
            ? css.aboveDropTarget
            : dropDirection === 'below'
              ? css.belowDropTarget
              : undefined
          : undefined
      }
      ref={(node) => drag(drop(node))}
      style={style}
      onClick={onClick}>
      {children}
    </li>
  );
};

export default DraggableListItem;
