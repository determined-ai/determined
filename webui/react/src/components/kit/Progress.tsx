import React from 'react';

import { floatToPercent } from 'components/kit/internal/string';
import Tooltip from 'components/kit/Tooltip';

import css from './Progress.module.scss';

export interface BarPart {
  bordered?: string;
  color: string; // css color
  label?: string;
  percent: number; // between 0-1
}

export interface Props {
  height?: number;
  inline?: boolean;
  parts: BarPart[];
  title?: string;
  tooltip?: string;
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

const Progress: React.FC<Props> = ({ height = 4, inline, parts, title, tooltip }: Props) => {
  const classes: string[] = [css.base];

  if (inline) classes.push(css.inline);

  const pbar = (
    <div className={classes.join(' ')}>
      <div className={css.bar} style={{ height: `calc(${height}px + var(--theme-density) * 1px)` }}>
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

  return (
    <>
      {title && <h3>{title}</h3>}
      {tooltip ? <Tooltip content={tooltip}>{pbar}</Tooltip> : pbar}
    </>
  );
};

export default Progress;
