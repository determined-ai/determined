import type { TooltipProps } from 'determined-ui/kit/Tooltip';
import React from 'react';

const { default: OriginalTooltip } = await vi.importActual<
  typeof import('determined-ui/kit/Tooltip')
>('determined-ui/kit/Tooltip');

const Tooltip: React.FC<TooltipProps> = (props: TooltipProps) => {
  return <OriginalTooltip {...props} mouseEnterDelay={0} />;
};
export default Tooltip;
