import React, { useMemo } from 'react';

import Icon, { Props as IconProps, IconSize } from 'components/kit/Icon';
import { stateToLabel } from 'constants/states';
import { CompoundRunState, JobState, RunState } from 'types';

interface Props {
  showTooltip?: boolean;
  state: CompoundRunState;
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
        return { backgroundColor, name: 'queued', opacity, title: stateToLabel(state) };
      case RunState.Starting:
      case RunState.Pulling:
        return { name: 'spin-bowtie', title: stateToLabel(state) };
      case RunState.Running:
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
      default:
        return { color: 'cancel', name: 'cancelled', title: 'Stopped' };
    }
  }, [backgroundColor, opacity, state]);

  return <Icon showTooltip={showTooltip} size={size} {...iconProps} />;
};

export default ExperimentIcons;
