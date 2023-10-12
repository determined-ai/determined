import React from 'react';

import Badge from 'components/kit/Badge';
import { StateOfUnion } from 'components/kit/Theme';
import { badgeColorFromState, stateToLabel } from 'constants/states';

interface StateBadgeProps {
  state: StateOfUnion;
}

export const StateBadge: React.FC<StateBadgeProps> = ({ state }: StateBadgeProps) => {
  return (
    <Badge
      badgeColor={badgeColorFromState(state)}
      dashed={state === 'POTENTIAL'}
      text={stateToLabel(state).toUpperCase()}
    />
  );
};
