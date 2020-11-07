import { Tooltip } from 'antd';
import React from 'react';

import { getStateColorCssVar } from 'themes';
import { CommandState, RunState } from 'types';
import { floatToPercent } from 'utils/string';

import css from './ProgressBar.module.scss';

export interface Props {
  barOnly?: boolean;
  percent: number;
  state: RunState | CommandState;
}

const defaultProps = { percent: 0 };

const ProgressBar: React.FC<Props> = ({ barOnly, percent, state }: Props) => {
  const classes = [ css.base ];
  const style = {
    backgroundColor: getStateColorCssVar(state),
    width: `${percent}%`,
  };

  if (barOnly) classes.push(css.barOnly);

  return (
    <Tooltip title={floatToPercent(percent / 100, 0)}>
      <div className={classes.join(' ')}>
        <div className={css.bar}>
          <span className={css.progress} style={style} />
        </div>
      </div>
    </Tooltip>
  );
};

ProgressBar.defaultProps = defaultProps;

export default ProgressBar;
