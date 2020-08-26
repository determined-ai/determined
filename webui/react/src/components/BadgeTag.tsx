import React, { PropsWithChildren } from 'react';

import Badge, { BadgeProps } from './Badge';
import css from './BadgeTag.module.scss';

export { BadgeType } from './Badge';

interface Props extends BadgeProps {
  preLabel?: React.ReactNode;
  label?: React.ReactNode;
}

const BadgeTag: React.FC<Props> = ({
  children,
  label,
  preLabel,
  ...props
}: PropsWithChildren<Props>) => {
  return (
    <span className={css.base}>
      {preLabel && <span className={css.preLabel}>{preLabel}</span>}
      <Badge {...props}>{children}</Badge>
      {label && <span className={css.label}>{label}</span>}
    </span>
  );
};

export default BadgeTag;
