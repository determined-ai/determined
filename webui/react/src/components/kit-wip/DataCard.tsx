import React, { ReactNode } from 'react';

interface DataCardProps {
}

const DataCardComponent: React.FC<DataCardProps> = (props: DataCardProps) => {
  return (
    <div {...props} />
  );
};
//OverviewStats
//ResourcePoolCard

export default DataCardComponent;
