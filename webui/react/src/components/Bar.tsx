import { Tooltip } from 'antd';
import React from 'react';

import { ShirtSize } from 'themes';
import { floatToPercent } from 'utils/string';

import css from './Bar.module.scss';

export interface BarPart {
  color: string; // css color
  label?: string;
  percent: number; // between 0-1
}

export interface Props {
  barOnly?: boolean;
  parts: BarPart[];
  size?: ShirtSize;
}

const partStyle = (part: BarPart) => {
  return {
    backgroundColor: part.color,
    width: floatToPercent(part.percent, 0),
  };
};

const Bar: React.FC<Props> = ({ barOnly, parts, size }: Props) => {
  const classes: string[] = [ css.base ];

  if (barOnly) classes.push(css.barOnly);

  return (
    <div className={classes.join(' ')}>
      <div
        className={css.bar}
        style={{ height: `var(--theme-sizes-layout-${size || ShirtSize.tiny})` }}>
        <div className={css.parts}>
          {parts.filter(part => part.percent !== 0 && !isNaN(part.percent)).map((part, idx) => {
            return (
              <Tooltip key={idx} title={part.label}>
                <li style={partStyle(part)} />
              </Tooltip>
            );
          })}
        </div>

      </div>
    </div>
  );
};

export default Bar;
