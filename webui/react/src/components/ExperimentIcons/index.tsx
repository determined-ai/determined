import { Tooltip } from 'antd';
import React, { CSSProperties, useMemo } from 'react';

import { stateToLabel } from 'constants/states';
import Icon from 'shared/components/Icon/Icon';
import { CompoundRunState, JobState, RunState } from 'types';

import Active from './Active';
import Queue from './Queue';
import Spinner from './Spinner';

interface Props {
  height?: CSSProperties['height'];
  isTooltipVisible?: boolean;
  state: CompoundRunState;
  width?: CSSProperties['width'];
}

const ExperimentIcons: React.FC<Props> = ({ state, height, width, isTooltipVisible = true }) => {
  const icon = useMemo(() => {
    const IconStyle: CSSProperties = { fontWeight: 900 };
    switch (state) {
      case JobState.SCHEDULED:
      case JobState.SCHEDULEDBACKFILLED:
      case JobState.QUEUED:
      case RunState.Queued:
        return <Queue height={height} width={width} />;
      case RunState.Starting:
      case RunState.Pulling:
        return <Spinner height={height} type="bowtie" width={width} />;
      case RunState.Running:
        return <Spinner height={height} type="shadow" width={width} />;
      case RunState.Paused:
        return <Icon name="pause" style={{ color: 'var(--theme-ix-cancel)', height, width }} />;
      case RunState.Completed:
        return (
          <Icon
            name="checkmark"
            style={{ height, width, ...IconStyle, color: 'var(--theme-status-success)' }}
          />
        );
      case RunState.Error:
      case RunState.Deleted:
      case RunState.Deleting:
      case RunState.DeleteFailed:
        return (
          <Icon
            name="error"
            style={{ height, width, ...IconStyle, color: 'var(--theme-status-error)' }}
          />
        );
      case RunState.Active:
      case RunState.Unspecified:
      case JobState.UNSPECIFIED:
        return <Active />;
      default:
        return (
          <Icon
            name="cancelled"
            style={{ height, width, ...IconStyle, color: 'var(--theme-ix-cancel)' }}
          />
        );
    }
  }, [height, state, width]);

  return (
    <>
      {isTooltipVisible ? (
        <Tooltip placement="bottom" title={stateToLabel(state)}>
          <span style={{ display: 'inherit' }}>{icon}</span>
        </Tooltip>
      ) : (
        <span style={{ display: 'inherit' }}>{icon}</span>
      )}
    </>
  );
};

export default ExperimentIcons;
