import React from 'react';

import { getStateColor } from 'themes';
import { CommandState, RunState } from 'types';

import css from './ProgressBar.module.scss';

interface Props {
  percent: number;
  state: RunState | CommandState;
  title?: string;
}

const defaultProps = {
  percent: 0,
};

const ProgressBar: React.FC<Props> = ({ percent, state, ...rest }: Props) => {
  const style = {
    backgroundColor: getStateColor(state),
    width: `${percent}%`,
  };

  return (
    <div className={css.base} {...rest}>
      <span className={css.progress} style={style} />
    </div>
  );
};

ProgressBar.defaultProps = defaultProps;

export default ProgressBar;
