import Icon, { Props as IconProps, IconSize } from 'hew/Icon';
import React, { useMemo } from 'react';

import { stateToLabel } from 'constants/states';
import { CommandState, CompoundRunState, JobState, RunState } from 'types';

interface Props {
  showTooltip?: boolean;
  state: CompoundRunState | CommandState;
  size?: IconSize;
  backgroundColor?: React.CSSProperties['backgroundColor'];
  opacity?: React.CSSProperties['opacity'];
}

const ExperimentIcons: React.FC<Props> = ({
  state,
  showTooltip = true,
  size,
  backgroundColor,
  opacity,
}) => {
  const iconProps: IconProps = useMemo(() => {
    switch (state) {
      case JobState.SCHEDULED:
      case JobState.SCHEDULEDBACKFILLED:
      case JobState.QUEUED:
      case RunState.Queued:
      case CommandState.Queued:
      case CommandState.Waiting:
        return { backgroundColor, name: 'queued', opacity, title: stateToLabel(state) };
      case RunState.Starting:
      case RunState.Pulling:
      case CommandState.Starting:
      case CommandState.Pulling:
        return { name: 'spin-bowtie', title: stateToLabel(state) };
      case RunState.Running:
      case CommandState.Running:
        return { name: 'spin-shadow', title: stateToLabel(state) };
      case RunState.Paused:
        return { color: 'cancel', name: 'pause', title: 'Paused' };
      case RunState.Completed:
        return { color: 'success', name: 'checkmark', title: 'Completed' };
      case RunState.Error:
      case RunState.Deleted:
      case RunState.Deleting:
      case RunState.DeleteFailed:
        return { color: 'error', name: 'error', title: 'Error' };
      case RunState.Active:
      case RunState.Unspecified:
      case JobState.UNSPECIFIED:
        return { name: 'active', title: stateToLabel(state) };
      case RunState.StoppingCanceled:
      case RunState.StoppingCompleted:
      case RunState.StoppingError:
      case RunState.StoppingKilled:
      case CommandState.Terminating:
        return { color: 'cancel', name: 'spin-shadow', title: stateToLabel(state) };
      case RunState.Canceled:
      case CommandState.Terminated:
        return { color: 'cancel', name: 'cancelled', title: 'Stopped' };
      default:
        return { color: 'cancel', name: 'exclamation-circle', title: 'Unknown State' };
    }
  }, [backgroundColor, opacity, state]);

  return <Icon showTooltip={showTooltip} size={size} {...iconProps} />;
};

export default ExperimentIcons;
