import React, { CSSProperties } from 'react';
import { useDrag, useDrop } from 'react-dnd';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';

import css from './DraggableListItem.module.scss';

interface Props {
  children: React.ReactNode;
  index: number;
  onClick: (event: React.MouseEvent) => void;
  onDrop: (indexOne: number, indexTwo: number) => void;
  style: CSSProperties;
}
interface DroppableItemProps {
  index: number
}

const DraggableTypes = { COLUMN: 'COLUMN' };

/* eslint-disable-next-line @typescript-eslint/ban-types */
const withDragAndDropProvider = <T extends Props>
(Component: React.FunctionComponent<T>) =>
  (props: T) => (
    <DndProvider backend={HTML5Backend}>
      <Component {...props} />
    </DndProvider>
  );

const DraggableListItem: React.FC<Props> = ({ index, style, onClick, children, onDrop }: Props) => {

  const [ { isOver }, drop ] = useDrop(
    () => ({
      accept: DraggableTypes.COLUMN,
      collect: (monitor) => ({
        canDrop: !!monitor.canDrop(),
        isOver: !!monitor.isOver(),
      }),
      drop: (item: DroppableItemProps) => {
        onDrop(index, item.index);
      },
    }),
    [],
  );

  const [ , drag ] = useDrag(() => ({
    item: { index },
    type: DraggableTypes.COLUMN,
  }));

  return (
    <span ref={drop}>
      <li
        className={isOver ? css.dropTarget : undefined}
        ref={drag}
        style={style}
        onClick={onClick}>
        {children}
      </li>
    </span>
  );
};
export default withDragAndDropProvider(DraggableListItem);
