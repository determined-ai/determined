import { GridCell } from '@glideapps/glide-data-grid';
import { MenuProps } from 'antd';
import { DropdownEvent } from 'hew/Dropdown';
import { MutableRefObject, useCallback, useEffect, useRef } from 'react';

// eslint-disable-next-line
function useOutsideClickHandler(ref: MutableRefObject<any>, handler: (event: Event) => void) {
  useEffect(() => {
    /**
     * Alert if clicked on outside of element
     */
    function handleClickOutside(event: Event) {
      if (
        ref.current &&
        !ref.current.contains(event.target) &&
        (!(event.target instanceof Element) || !event.target.className.includes('ant-dropdown'))
      ) {
        handler(event);
      }
    }
    // Bind the event listener
    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      // Unbind the event listener on clean up
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [ref, handler]);
}

export type ContextMenuCompleteHandlerProps<CompleteAction, CompleteData> = (
  action: CompleteAction,
  id: number,
  data?: Partial<CompleteData>,
) => void;

export interface ContextMenuProps<RowData, CompleteAction, CompleteData> extends MenuProps {
  cell?: GridCell;
  rowData: RowData;
  link?: string;
  onClose: (e?: DropdownEvent | Event) => void;
  onComplete?: ContextMenuCompleteHandlerProps<CompleteAction, CompleteData>;
  onVisibleChange?: (visible: boolean) => void;
  open: boolean;
  renderContextMenuComponent?: (
    props: ContextMenuComponentProps<RowData, CompleteAction, CompleteData>,
  ) => JSX.Element;
  x: number;
  y: number;
}

export type ContextMenuComponentProps<RowData, CompleteAction, CompleteData> = Omit<
  ContextMenuProps<RowData, CompleteAction, CompleteData>,
  'renderContextMenuComponent' | 'x' | 'y'
>;

export function ContextMenu<RowData, CompleteAction, CompleteData>({
  cell,
  rowData,
  link,
  onClose,
  onComplete,
  open,
  renderContextMenuComponent,
  x,
  y,
}: ContextMenuProps<RowData, CompleteAction, CompleteData>): JSX.Element {
  const containerRef = useRef(null);
  useOutsideClickHandler(containerRef, onClose);

  const handleComplete = useCallback(
    (action: CompleteAction, id: number, data?: Partial<CompleteData>) => {
      onComplete?.(action, id, data);
      onClose();
    },
    [onClose, onComplete],
  );

  const handleVisibleChange = useCallback(() => onClose(), [onClose]);

  return (
    <div
      ref={containerRef}
      style={{
        left: x,
        position: 'fixed',
        top: y,
        zIndex: 10,
      }}>
      {renderContextMenuComponent?.({
        cell,
        link,
        onClose,
        onComplete: handleComplete,
        onVisibleChange: handleVisibleChange,
        open,
        rowData,
      })}
    </div>
  );
}
