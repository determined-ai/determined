import React, { CSSProperties, PropsWithChildren } from 'react';

import { getStateColor } from 'themes';
import { CommandState, RunState } from 'types';
import { stateToLabel } from 'utils/types';

import css from './Badge.module.scss';

export enum BadgeType {
  Default,
  Id,
  State,
}

interface Props {
  state?: RunState | CommandState;
  type?: BadgeType;
}

const defaultProps = {
  state: RunState.Active,
  type: BadgeType.Default,
};

const Badge: React.FC<Props> = ({ state, type, children }: PropsWithChildren<Props>) => {
  const classes = [ css.base ];
  const style: CSSProperties = {};

  if (type === BadgeType.State) {
    classes.push(css.state);
    style.backgroundColor = getStateColor(state);
  } else if (type === BadgeType.Id) {
    classes.push(css.id);
  }

  return (
    <span className={classes.join(' ')} style={style}>
      {type === BadgeType.State && state ? stateToLabel(state) : children}
    </span>
  );
};

Badge.defaultProps = defaultProps;

export default Badge;
