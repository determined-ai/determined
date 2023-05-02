import React from 'react';

import Tooltip from 'components/kit/Tooltip';

import Badge, { BadgeProps } from './Badge';
import css from './BadgeTag.module.scss';

export interface Props extends BadgeProps {
  children?: React.ReactNode;
  label?: React.ReactNode;
  preLabel?: React.ReactNode;
}

const TOOLTIP_DELAY = 1.0;

const BadgeTag: React.FC<Props> = ({ children, label, preLabel, ...props }: Props) => {
  return (
    <span className={css.base}>
      {preLabel && (
        <Tooltip content={label} mouseEnterDelay={TOOLTIP_DELAY}>
          <span className={css.preLabel}>{preLabel}</span>
        </Tooltip>
      )}
      <Badge {...props}>{children}</Badge>
      {label && (
        <Tooltip content={label} mouseEnterDelay={TOOLTIP_DELAY}>
          <span className={css.label}>{label}</span>
        </Tooltip>
      )}
    </span>
  );
};

export default BadgeTag;
