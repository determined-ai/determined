import type { TooltipProps } from 'determined-ui/Tooltip';
import React from 'react';

const { default: OriginalTooltip } = await vi.importActual<typeof import('determined-ui/Tooltip')>(
  'determined-ui/Tooltip',
);

const Tooltip: React.FC<TooltipProps> = (props: TooltipProps) => {
  return <OriginalTooltip {...props} mouseEnterDelay={0} />;
};
export default Tooltip;
