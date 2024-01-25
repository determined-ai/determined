import { GridCell } from '@glideapps/glide-data-grid';
import { MenuProps } from 'antd';
import { DropdownEvent } from 'hew/Dropdown';
import React, { useCallback, useRef } from 'react';

import ExperimentActionDropdown from 'components/ExperimentActionDropdown';
import { ExperimentAction, ExperimentItem, ProjectExperiment } from 'types';

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
