import { GridCell } from '@hpe.com/glide-data-grid';
import { MenuProps } from 'antd';
import React, { MutableRefObject, useCallback, useEffect, useRef } from 'react';

import ExperimentActionDropdown from 'components/ExperimentActionDropdown';
import { DropdownEvent } from 'components/kit/Dropdown';
import { ExperimentAction, ExperimentItem, ProjectExperiment } from 'types';

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
        !(event.target ? (event.target as Element) : null)?.classList?.contains(
          'ant-dropdown-menu-title-content',
        )
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

export interface TableContextMenuProps extends MenuProps {
  cell?: GridCell;
  experiment: ProjectExperiment;
  link?: string;
  onClose: (e?: DropdownEvent | Event) => void;
  onComplete?: (action: ExperimentAction, id: number, data?: Partial<ExperimentItem>) => void;
  open: boolean;
  x: number;
  y: number;
}

export const TableContextMenu: React.FC<TableContextMenuProps> = ({
  cell,
  experiment,
  link,
  onClose,
  onComplete,
  open,
  x,
  y,
}) => {
  const containerRef = useRef(null);
  useOutsideClickHandler(containerRef, onClose);

  const handleComplete = useCallback(
    (action: ExperimentAction, id: number, data?: Partial<ExperimentItem>) => {
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
      <ExperimentActionDropdown
        cell={cell}
        experiment={experiment}
        link={link}
        makeOpen={open}
        onComplete={handleComplete}
        onLink={onClose}
        onVisibleChange={handleVisibleChange}>
        <div />
      </ExperimentActionDropdown>
    </div>
  );
};
