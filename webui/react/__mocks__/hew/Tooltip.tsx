import type { TooltipProps } from 'hew/Tooltip';
import React from 'react';

const { default: OriginalTooltip } = await vi.importActual<typeof import('hew/Tooltip')>(
  'hew/Tooltip',
);

const Tooltip: React.FC<TooltipProps> = (props: TooltipProps) => {
  return <OriginalTooltip {...props} mouseEnterDelay={0} />;
};
export default Tooltip;
