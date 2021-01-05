import { Tooltip } from 'antd';
import React, { CSSProperties, PropsWithChildren } from 'react';

import { getStateColorCssVar } from 'themes';
import { CheckpointState, CommandState, ResourceState, RunState } from 'types';
import { stateToLabel } from 'utils/types';

import css from './Badge.module.scss';

export enum BadgeType {
  Custom,
  Default,
  Header,
  Id,
  State,
}

export interface BadgeProps {
  bgColor?: string; // background color for custom badge.
  fgColor?: string; // foreground color for custom badge.
  state?: RunState | CommandState | CheckpointState | ResourceState;
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
  } else if (type === BadgeType.Header) {
    classes.push(css.header);
  } else if (type === BadgeType.Custom) {
    style.color = props.fgColor;
    style.backgroundColor = props.bgColor;
  }

  const badge = <span className={classes.join(' ')} style={style}>
    {type === BadgeType.State && state ? stateToLabel(state) : props.children}
  </span>;

  return tooltip ? <Tooltip title={tooltip}>{badge}</Tooltip> : badge;
};

export default Badge;
