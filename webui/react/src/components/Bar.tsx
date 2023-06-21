import React from 'react';

import Tooltip from 'components/kit/Tooltip';
import { ShirtSize } from 'themes';
import { floatToPercent } from 'utils/string';

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

const partStyle = (part: BarPart) => {
  let style = {
    backgroundColor: part.color,
    borderColor: 'var(--theme-float-border)',
    borderStyle: 'none',
    borderWidth: 1,
    width: floatToPercent(part.percent, 0),
  };

  if (part.bordered) {
    style = { ...style, borderStyle: 'dashed' };
  }

  return style;
};

const sizeMap = {
  [ShirtSize.Small]: '4px',
  [ShirtSize.Medium]: '12px',
  [ShirtSize.Large]: '24px',
};

const Bar: React.FC<Props> = ({ barOnly, inline, parts, size = ShirtSize.Small }: Props) => {
  const classes: string[] = [css.base];

  if (barOnly) classes.push(css.barOnly);
  if (inline) classes.push(css.inline);

  return (
    <div className={classes.join(' ')}>
      <div
        className={css.bar}
        style={{ height: `calc(${sizeMap[size]} + var(--theme-density) * 1px)` }}>
        <div className={css.parts}>
          {parts
            .filter((part) => part.percent !== 0 && !isNaN(part.percent))
            .map((part, idx) => (
              <Tooltip content={part.label} key={idx}>
                <li style={partStyle(part)} />
              </Tooltip>
            ))}
        </div>
      </div>
    </div>
  );
};

export default Bar;
