import { Tooltip } from 'antd';
import React from 'react';

import { floatToPercent } from 'shared/utils/string';
import { ShirtSize } from 'themes';

import css from './Bar.module.scss';

export interface BarPart {
  bordered?: string;
  color: string; // css color
  label?: string;
  percent: number; // between 0-1
}

export interface Props {
  barOnly?: boolean;
  inline?: boolean;
  parts: BarPart[];
  size?: ShirtSize;
}

const sizeMap = {
  [ShirtSize.small]: '4px',
  [ShirtSize.medium]: '12px',
  [ShirtSize.large]: '24px',
};

const partStyle = (part: BarPart) => ({
  backgroundColor: part.color,
  borderStyle: part.bordered ? 'dashed dashed dashed none' : 'none',
  width: floatToPercent(part.percent, 0),
});

const Bar: React.FC<Props> = ({ barOnly, inline, parts, size = ShirtSize.small }: Props) => {
  const classes: string[] = [ css.base ];

  if (barOnly) classes.push(css.barOnly);
  if (inline) classes.push(css.inline);

  return (
    <div className={classes.join(' ')}>
      <div
        className={css.bar}
        style={{ height: `calc(${sizeMap[size]} + var(--theme-density) * 1px)` }}>
        <div className={css.parts}>
          {parts.filter(part => part.percent !== 0 && !isNaN(part.percent)).map((part, idx) => (
            <Tooltip key={idx} title={part.label}>
              <li style={partStyle(part)} />
            </Tooltip>
          ))}
        </div>

      </div>
    </div>
  );
};

export default Bar;
