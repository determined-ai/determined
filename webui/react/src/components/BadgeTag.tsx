import { Tooltip } from 'antd';
import React, { PropsWithChildren } from 'react';

import Badge, { BadgeProps } from './Badge';
import css from './BadgeTag.module.scss';

export { BadgeType } from './Badge';

interface Props extends BadgeProps {
  tag?: React.ReactNode;
  preTag?: React.ReactNode;
}

const TOOLTIP_DELAY = 1.0;

const BadgeTag: React.FC<Props> = ({
  children,
  tag,
  preTag,
  ...props
}: PropsWithChildren<Props>) => {
  return (
    <span className={css.base}>
      {preTag && (
        <Tooltip mouseEnterDelay={TOOLTIP_DELAY} title={tag}>
          <span className={css.preTag}>{preTag}</span>
        </Tooltip>
      )}
      <Badge {...props}>{children}</Badge>
      {tag && (
        <Tooltip mouseEnterDelay={TOOLTIP_DELAY} title={tag}>
          <span className={css.tag}>{tag}</span>
        </Tooltip>
      )}
    </span>
  );
};

export default BadgeTag;
