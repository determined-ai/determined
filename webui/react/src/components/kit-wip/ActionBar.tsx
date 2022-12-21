import React, { ReactNode } from 'react';

interface ActionBarProps {
}

const ActionBarComponent: React.FC<ActionBarProps> = (props: ActionBarProps) => {
  return (
    <div {...props} />
  );
};

// ExperimentDetailsHeader

export default ActionBarComponent;
