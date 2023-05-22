import React, { useMemo } from 'react';

import Pivot, { TabItem } from 'components/kit/Pivot';

import css from './ComparisonView.module.scss';

interface Props {
  children?: React.ReactElement;
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

  const viewClasses = [css.comparisonView];
  if (open) viewClasses.push(css.open);

  return (
    <div className={css.base}>
      {children}
      <div className={viewClasses.join(' ')}>
        <Pivot items={tabs} />
      </div>
    </div>
  );
};

export default ComparisonView;
