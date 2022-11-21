import { Tooltip } from 'antd';
import React, { useMemo } from 'react';

import { runStateToLabel } from 'constants/states';
import Icon from 'shared/components/Icon/Icon';
import { RunState } from 'types';

import Active from './Active';
import Queue from './Queue';
import Spinner from './Spinner';

interface Props {
  state: RunState;
}

const ExperimentIcons: React.FC<Props> = ({ state }) => {
  const icon = useMemo(() => {
    switch (state) {
      case RunState.Queued:
        return <Queue />;
      case RunState.Starting:
      case RunState.Pulling:
        return <Spinner type="bowtie" />;
      case RunState.Running:
        return <Spinner type="shadow" />;
      case RunState.Paused:
        return <Icon name="pause" style={{ color: 'var(--theme-ix-cancel)' }} />;
      case RunState.Completed:
        return <Icon name="checkmark" style={{ color: 'var(--theme-status-success)' }} />;
      case RunState.Error:
      case RunState.Deleted:
      case RunState.Deleting:
      case RunState.DeleteFailed:
        return <Icon name="error" style={{ color: 'var(--theme-status-error)' }} />;
      case RunState.Active:
      case RunState.Unspecified:
        return <Active />;
      default:
        return <Icon name="cancelled" style={{ color: 'var(--theme-ix-cancel)' }} />;
    }
  }, [state]);
  return (
    <Tooltip title={runStateToLabel[state]}>
      <span>{icon}</span>
    </Tooltip>
  );
};

export default ExperimentIcons;
