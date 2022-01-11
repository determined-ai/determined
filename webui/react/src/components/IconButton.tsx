import { Button, Tooltip } from 'antd';
import { ButtonType } from 'antd/es/button';
import { TooltipPlacement } from 'antd/es/tooltip';
import React, { useCallback } from 'react';

import Icon, { IconSize } from 'components/Icon';

import css from './IconButton.module.scss';

interface Props {
  className?: string;
  icon: string;
  iconSize?: IconSize;
  label: string;
  onClick?: (event: React.MouseEvent) => void;
  tooltipPlacement?: TooltipPlacement;
  type?: ButtonType;
}

const IconButton: React.FC<Props> = ({
  className,
  icon,
  iconSize = 'medium',
  label,
  onClick,
  tooltipPlacement = 'top',
  type,
}: Props) => {
  const classes = [ css.base ];

  if (className) classes.push(className);

  const handleClick = useCallback((e: React.MouseEvent) => {
    if (onClick) onClick(e);
  }, [ onClick ]);

  return (
    <Tooltip placement={tooltipPlacement} title={label}>
      <Button
        aria-label={label}
        className={classes.join(' ')}
        type={type}
        onClick={handleClick}>
        <Icon name={icon} size={iconSize} />
      </Button>
    </Tooltip>
  );
};

export default IconButton;
