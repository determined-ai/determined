import { number } from 'fp-ts';
import React, { CSSProperties } from 'react';
import {useDrag, useDrop} from 'react-dnd';
import { DndProvider } from 'react-dnd'
import { HTML5Backend } from 'react-dnd-html5-backend'

import css from './DraggableListItem.module.scss';

interface Props {
  style: CSSProperties;
  onClick: (event: React.MouseEvent) => {};
  children: React.ReactChildren;
  onDrop: (indexOne: number, indexTwo: number) => {};
  index: number;
}

interface DroppableItemProps {
  index: number
}

const withDragAndDropProvider = (Component: React.FunctionComponent<any>) => (props: any) =>{
  return (
    <DndProvider backend={HTML5Backend}>
        <Component {...props}/>
    </DndProvider>
  )
}

const DraggableTypes = {
  COLUMN: 'COLUMN'
}


  const DraggableListItem: React.FC<Props>= ({index, style ,onClick, children, onDrop }: Props) => {

    const [{ isOver }, drop] = useDrop(
      () => ({
        accept: DraggableTypes.COLUMN,
        drop: (item: DroppableItemProps) => {
          onDrop(index, item.index)
        },
        collect: (monitor) => ({
          isOver: !!monitor.isOver(),
          canDrop: !!monitor.canDrop()
        })
      }),
      []
    )

    const [, drag] = useDrag(() => ({
      type: DraggableTypes.COLUMN,
      item: {index},
    }))
    
      return (
        <span ref={drop}>
          <li ref={drag} className={isOver ? css.dropTarget: undefined} style={style} onClick={onClick}>
              {children}
          </li>
        </span>
      )
    }
  export default withDragAndDropProvider(DraggableListItem);