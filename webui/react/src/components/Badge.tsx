import { Tooltip } from 'antd';
import React, { CSSProperties, PropsWithChildren } from 'react';

import { getStateColorCssVar } from 'themes';
import { CheckpointState, CommandState, RunState } from 'types';
import { stateToLabel } from 'utils/types';

import css from './Badge.module.scss';

export enum BadgeType {
  Default,
  Id,
  State,
}

export interface BadgeProps {
  state?: RunState | CommandState | CheckpointState;
  tooltip?: string;
  type?: BadgeType;
}

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
  } else if (type === BadgeType.Id) {
    classes.push(css.id);
  }

  const badge = <span className={classes.join(' ')} style={style}>
    {type === BadgeType.State && state ? stateToLabel(state) : props.children}
  </span>;

  return tooltip ? <Tooltip title={tooltip}>{badge}</Tooltip> : badge;
};

export default Badge;
