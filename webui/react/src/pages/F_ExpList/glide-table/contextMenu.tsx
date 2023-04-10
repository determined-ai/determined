import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Menu, MenuProps } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';
import React, { MutableRefObject, useCallback, useEffect, useRef } from 'react';

import ExperimentActionDropdown from 'components/ExperimentActionDropdown';
import usePermissions from 'hooks/usePermissions';
import {
  activateExperiment,
  archiveExperiment,
  cancelExperiment,
  deleteExperiment,
  killExperiment,
  openOrCreateTensorBoard,
  pauseExperiment,
  unarchiveExperiment,
} from 'services/api';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { capitalize } from 'shared/utils/string';
import { ExperimentAction as Action, ProjectExperiment } from 'types';
import { modal } from 'utils/dialogApi';
import handleError from 'utils/error';
import { getActionsForExperiment } from 'utils/experiment';
import { openCommandResponse } from 'utils/wait';

const dropdownActions = [
  // Action.SwitchPin, requires settings
  Action.Activate,
  Action.Pause,
  Action.Archive,
  Action.Unarchive,
  Action.Cancel,
  Action.Kill,
  Action.Move,
  Action.OpenTensorBoard,
  Action.HyperparameterSearch,
  Action.Delete,
];

// eslint-disable-next-line
function useOutsideClickHandler(ref: MutableRefObject<any>, handler: (event: Event) => void) {
  useEffect(() => {
    /**
     * Alert if clicked on outside of element
     */
    function handleClickOutside(event: Event) {
      if (ref.current && !ref.current.contains(event.target)) {
        handler(event);
      }
    }
    // Bind the event listener
    document.addEventListener('mouseup', handleClickOutside);
    return () => {
      // Unbind the event listener on clean up
      document.removeEventListener('mouseup', handleClickOutside);
    };
  }, [ref, handler]);
}

export interface TableContextMenuProps extends MenuProps {
  fetchExperiments: () => void;
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

  const onComplete = useCallback(() => {
    fetchExperiments();
  }, [fetchExperiments]);

  // <Menu items={menuItems} onClick={handleMenuClick} />

  return (
    <div
      ref={containerRef}
      style={{
        border: 'solid 1px gold',
        display: !open ? 'none' : undefined,
        left: x,
        position: 'fixed',
        top: y,
        width: 200,
      }}>
      <ExperimentActionDropdown
        experiment={experiment}
        onComplete={onComplete} />
    </div>
  );
};
