import { MenuProps } from 'antd';
import React, { MutableRefObject, useCallback, useEffect, useRef } from 'react';

import ExperimentActionDropdown from 'components/ExperimentActionDropdown';
import { ProjectExperiment } from 'types';

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
  fetchExperiments: () => Promise<void>;
  open: boolean;
  experiment: ProjectExperiment;
  handleClose: (e?: Event) => void;
  x: number;
  y: number;
}

export const TableContextMenu: React.FC<TableContextMenuProps> = ({
  experiment,
  fetchExperiments,
  handleClose,
  open,
  x,
  y,
}) => {
  const containerRef = useRef(null);
  useOutsideClickHandler(containerRef, handleClose);

  const onComplete = useCallback(async () => {
    await fetchExperiments();
    handleClose();
  }, [fetchExperiments, handleClose]);

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
        experiment={experiment}
        makeOpen={open}
        onComplete={onComplete}
        onVisibleChange={onComplete}>
        <div />
      </ExperimentActionDropdown>
    </div>
  );
};
