import React from 'react';

import type { TooltipProps } from 'components/kit/Tooltip';
const { default: OriginalTooltip } = await vi.importActual<typeof import('components/kit/Tooltip')>(
  'components/kit/Tooltip',
);

const Tooltip = (props: TooltipProps) => {
  return <OriginalTooltip {...props} mouseEnterDelay={0} />;
};
export default Tooltip;
