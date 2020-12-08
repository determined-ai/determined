import { Tooltip } from 'antd';
import React from 'react';

import { floatToPercent } from 'utils/string';

import css from './ProgressBar.module.scss';

interface BarPart {
  percent: number; // between 0-1
  color: string; // css color
  label: string;
}

export interface Props {
  barOnly?: boolean;
  parts: BarPart[]
}

const partStyle = (part: BarPart) => {
  return {
    backgroundColor: part.color,
    width: floatToPercent(part.percent),
  };
};

const SlotAllocationBar: React.FC<Props> = ({ barOnly, parts }: Props) => {
  const classes: string[] = [ css.base ];

  if (barOnly) classes.push(css.barOnly);

  return (
    <div className={classes.join(' ')}>
      <div className={css.bar}>
        <ol>
          {parts.map(part => {
            return (
              <Tooltip key={part.label} title={floatToPercent(part.percent)}>
                <li style={partStyle(part)} />
              </Tooltip>
            );
          })}
        </ol>

      </div>
    </div>
  );
};

export default SlotAllocationBar;
