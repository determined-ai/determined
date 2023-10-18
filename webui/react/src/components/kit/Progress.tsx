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
  showLegend?: boolean;
  title?: string;
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

const Progress: React.FC<Props> = ({ height = 4, inline, parts, showLegend, title }: Props) => {
  const classes: string[] = [css.base];

  if (inline) classes.push(css.inline);

  return (
    <>
      {title && <h5 className={css.title}>{title}</h5>}
      <div className={classes.join(' ')}>
        <div
          className={css.bar}
          style={{ height: `calc(${height}px + var(--theme-density) * 1px)` }}>
          <div className={css.parts}>
            {parts
              .filter((part) => part.percent !== 0 && !isNaN(part.percent))
              .map((part, idx) => (
                <Tooltip content={!showLegend && part.label} key={idx}>
                  <li
                    style={{
                      ...partStyle(part),
                      cursor: !showLegend && part.label ? 'pointer' : '',
                    }}
                  />
                </Tooltip>
              ))}
          </div>
        </div>
      </div>
      {showLegend && (
        <div className={css.legendContainer}>
          {parts
            .filter((part) => part.percent !== 0 && !isNaN(part.percent))
            .map((part, idx) => (
              <li className={css.legendItem} key={idx}>
                <span className={css.colorButton} style={partStyle(part)}>
                  -
                </span>
                {part.label} ({(part.percent * 100).toFixed(1)}%)
              </li>
            ))}
        </div>
      )}
    </>
  );
};

export default Progress;
