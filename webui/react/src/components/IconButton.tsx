import { Button } from 'antd';
import { ButtonType } from 'antd/es/button';
import { TooltipPlacement } from 'antd/es/tooltip';
import React, { useCallback } from 'react';

import Tooltip from 'components/kit/Tooltip';
import Icon, { IconSize } from 'shared/components/Icon/Icon';

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
  const handleClick = useCallback((e: React.MouseEvent) => onClick?.(e), [onClick]);

  return (
    <Tooltip placement={tooltipPlacement} title={label}>
      <Button
        aria-label={label}
        className={className}
        style={{ height: 'fit-content', paddingBottom: 0 }}
        type={type}
        onClick={handleClick}>
        <Icon name={icon} size={iconSize} />
      </Button>
    </Tooltip>
  );
};

export default IconButton;
