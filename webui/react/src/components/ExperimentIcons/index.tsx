import React, { CSSProperties, useMemo } from 'react';

import Icon from 'components/kit/Icon';
import Tooltip from 'components/kit/Tooltip';
import { stateToLabel } from 'constants/states';
import { CompoundRunState, JobState, RunState } from 'types';

import Active from './Active';
import Queue from './Queue';
import Spinner from './Spinner';

interface Props {
  showTooltip?: boolean;
  state: CompoundRunState;
  style?: CSSProperties;
}

const ExperimentIcons: React.FC<Props> = ({ state, style, showTooltip = true }) => {
  const icon = useMemo(() => {
    switch (state) {
      case JobState.SCHEDULED:
      case JobState.SCHEDULEDBACKFILLED:
      case JobState.QUEUED:
      case RunState.Queued:
        return <Queue style={style} />;
      case RunState.Starting:
      case RunState.Pulling:
        return <Spinner type="bowtie" />;
      case RunState.Running:
        return <Spinner type="shadow" />;
      case RunState.Paused:
        return <Icon color="cancel" name="pause" title="Paused" />;
      case RunState.Completed:
        return <Icon color="success" name="checkmark" title="Completed" />;
      case RunState.Error:
      case RunState.Deleted:
      case RunState.Deleting:
      case RunState.DeleteFailed:
        return <Icon color="error" name="error" title="Error" />;
      case RunState.Active:
      case RunState.Unspecified:
      case JobState.UNSPECIFIED:
        return <Active />;
      default:
        return <Icon color="cancel" name="cancelled" title="Stopped" />;
    }
  }, [state, style]);

  return (
    <>
      {showTooltip ? (
        <Tooltip content={stateToLabel(state)} placement="bottom">
          <div style={{ display: 'flex' }}>{icon}</div>
        </Tooltip>
      ) : (
        <div style={{ display: 'flex' }}>{icon}</div>
      )}
    </>
  );
};

export default ExperimentIcons;
