import React, { ReactNode } from 'react';

interface ActionBarProps {
}

const ActionBar: React.FC<ActionBarProps> = (props: ActionBarProps) => {
  return (
    <div {...props} />
  );
};

// ExperimentDetailsHeader

export default ActionBar;
