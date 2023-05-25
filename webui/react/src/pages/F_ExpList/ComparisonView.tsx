import React, { useMemo } from 'react';

import Pivot, { TabItem } from 'components/kit/Pivot';
import SplitPane from 'components/SplitPane';

interface Props {
  children: React.ReactElement;
  open: boolean;
}

const ComparisonView: React.FC<Props> = ({ children, open }) => {
  const tabs: TabItem[] = useMemo(() => {
    return [
      { key: 'metrics', label: 'Metrics' },
      { key: 'hyperparameters', label: 'Hyperparameters' },
      { key: 'configurations', label: 'Configurations' },
    ];
  }, []);

  return (
    <div>
      <SplitPane open={open}>
        {children}
        <Pivot items={tabs} />
      </SplitPane>
    </div>
  );
};

export default ComparisonView;
