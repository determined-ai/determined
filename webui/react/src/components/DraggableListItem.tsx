import React, { CSSProperties } from 'react';
import { DndProvider, useDrag, useDrop } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';

import css from 'components/DraggableListItem.module.scss';

interface Props {
  children: React.ReactNode;
  columnName: string;
  index: number;
  onClick: (event: React.MouseEvent) => void;
  onDrop: (column: string, newNeighborColumnName: string) => void;
  style: CSSProperties;
}
interface DroppableItemProps {
  columnName: string;
  index: number;
}

const DraggableTypes = { COLUMN: 'COLUMN' };

// prettier-ignore
/* eslint-disable-next-line @typescript-eslint/ban-types */
const withDragAndDropProvider = <T extends {}>(Component: React.FunctionComponent<T>) =>
  (props: T) =>
    (
      <DndProvider backend={HTML5Backend}>
        <Component {...props} />
      </DndProvider>
    );

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
        canDrop: !!monitor.canDrop(),
        dropDirection: monitor.getItem()?.index > index ? 'above' : 'below',
        isOver: !!monitor.isOver(),
      }),
      drop: (item: DroppableItemProps) => {
        onDrop(item.columnName, columnName);
      },
    }),
    [],
  );

  const [, drag] = useDrag(() => ({
    item: { columnName, index },
    type: DraggableTypes.COLUMN,
  }));

  return (
    <span ref={drop}>
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
        ref={drag}
        style={style}
        onClick={onClick}>
        {children}
      </li>
    </span>
  );
};
export default withDragAndDropProvider(DraggableListItem);
