import type { Props } from 'hew/Tooltip';
import React from 'react';

const { default: OriginalTooltip } = await vi.importActual<typeof import('hew/Tooltip')>(
  'hew/Tooltip',
);

const Tooltip: React.FC<Props> = (props: Props) => {
  return <OriginalTooltip {...props} mouseEnterDelay={0} />;
};
export default Tooltip;
