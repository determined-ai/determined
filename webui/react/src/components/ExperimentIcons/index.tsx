import { Tooltip } from 'antd';
import React, { CSSProperties, useMemo } from 'react';

import { stateToLabel } from 'constants/states';
import Icon from 'shared/components/Icon/Icon';
import { CompoundRunState, JobState, RunState } from 'types';

import Active from './Active';
import Queue from './Queue';
import Spinner from './Spinner';

interface Props {
  isTooltipVisible?: boolean;
  state: CompoundRunState;
  style?: CSSProperties;
}

const ExperimentIcons: React.FC<Props> = ({ state, style, isTooltipVisible = true }) => {
  const icon = useMemo(() => {
    const IconStyle: CSSProperties = { fontWeight: 900 };
    switch (state) {
      case JobState.SCHEDULED:
      case JobState.SCHEDULEDBACKFILLED:
      case JobState.QUEUED:
      case RunState.Queued:
        return <Queue style={style} />;
      case RunState.Starting:
      case RunState.Pulling:
        return <Spinner style={style} type="bowtie" />;
      case RunState.Running:
        return <Spinner style={style} type="shadow" />;
      case RunState.Paused:
        return <Icon name="pause" style={{ ...style, color: 'var(--theme-ix-cancel)' }} />;
      case RunState.Completed:
        return (
          <Icon
            name="checkmark"
            style={{ ...style, ...IconStyle, color: 'var(--theme-status-success)' }}
          />
        );
      case RunState.Error:
      case RunState.Deleted:
      case RunState.Deleting:
      case RunState.DeleteFailed:
        return (
          <Icon
            name="error"
            style={{ ...style, ...IconStyle, color: 'var(--theme-status-error)' }}
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
            style={{ ...style, ...IconStyle, color: 'var(--theme-ix-cancel)' }}
          />
        );
    }
  }, [state, style]);

  return (
    <>
      {isTooltipVisible ? (
        <Tooltip placement="bottom" title={stateToLabel(state)}>
          <div style={{ display: 'flex' }}>{icon}</div>
        </Tooltip>
      ) : (
        <div style={{ display: 'flex' }}>{icon}</div>
      )}
    </>
  );
};

export default ExperimentIcons;
