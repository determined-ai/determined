import { Tooltip } from 'antd';
import React, { CSSProperties, useMemo } from 'react';

import { stateToLabel } from 'constants/states';
import Icon from 'shared/components/Icon/Icon';
import { CompoundRunState, JobState, RunState } from 'types';

import Active from './Active';
import Queue from './Queue';
import Spinner from './Spinner';

interface Props {
  state: CompoundRunState;
}

const ExperimentIcons: React.FC<Props> = ({ state }) => {
  const icon = useMemo(() => {
    const IconStyle: CSSProperties = { fontWeight: 900 };
    switch (state) {
      case JobState.SCHEDULED:
      case JobState.SCHEDULEDBACKFILLED:
      case JobState.QUEUED:
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
        return (
          <Icon name="checkmark" style={{ ...IconStyle, color: 'var(--theme-status-success)' }} />
        );
      case RunState.Error:
      case RunState.Deleted:
      case RunState.Deleting:
      case RunState.DeleteFailed:
        return <Icon name="error" style={{ ...IconStyle, color: 'var(--theme-status-error)' }} />;
      case RunState.Active:
      case RunState.Unspecified:
      case JobState.UNSPECIFIED:
        return <Active />;
      default:
        return <Icon name="cancelled" style={{ ...IconStyle, color: 'var(--theme-ix-cancel)' }} />;
    }
  }, [state]);
  return (
    <Tooltip title={stateToLabel(state)}>
      <span style={{ display: 'inherit' }}>{icon}</span>
    </Tooltip>
  );
};

export default ExperimentIcons;
