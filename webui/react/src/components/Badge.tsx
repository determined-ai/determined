import { Tooltip } from 'antd';
import React, { CSSProperties, PropsWithChildren } from 'react';

import {
  checkpointStateToLabel, commandStateToLabel, resourceStateToLabel,
  runStateToLabel, slotStateToLabel,
} from 'constants/states';
import { getStateColorCssVar } from 'themes';
import { CheckpointState, CommandState, ResourceState, RunState, SlotState } from 'types';

import css from './Badge.module.scss';

export enum BadgeType {
  Default,
  Header,
  Id,
  State,
}

export interface BadgeProps {
  state?: RunState | CommandState | CheckpointState | ResourceState | SlotState;
  tooltip?: string;
  type?: BadgeType;
}

const stateToLabel = (
  state: RunState | CommandState | CheckpointState | ResourceState | SlotState,
): string => {
  return runStateToLabel[state as RunState]
    || commandStateToLabel[state as CommandState]
    || resourceStateToLabel[state as ResourceState]
    || checkpointStateToLabel[state as CheckpointState]
    || slotStateToLabel[state as SlotState];
};

const Badge: React.FC<BadgeProps> = ({
  state = RunState.Active,
  tooltip,
  type = BadgeType.Default,
  ...props
}: PropsWithChildren<BadgeProps>) => {
  const classes = [ css.base ];
  const style: CSSProperties = {};

  if (type === BadgeType.State) {
    classes.push(css.state);
    style.backgroundColor = getStateColorCssVar(state);
    if (state === SlotState.Free) {
      style.color = '#234b65';
    }
  } else if (type === BadgeType.Id) {
    classes.push(css.id);
  } else if (type === BadgeType.Header) {
    classes.push(css.header);
  }

  const badge = <span className={classes.join(' ')} style={style}>
    {props.children ? props.children : type === BadgeType.State && state && stateToLabel(state)}
  </span>;

  return tooltip ? <Tooltip title={tooltip}>{badge}</Tooltip> : badge;
};

export default Badge;
