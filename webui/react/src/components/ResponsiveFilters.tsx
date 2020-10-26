import { Button } from 'antd';
import React from 'react';

import Dropdown, { Placement } from 'components/Dropdown';

import css from './ResponsiveFilters.module.scss';

interface Props {
  children: React.ReactNode;
  hasFiltersApplied?: boolean;
}

const ResponsiveFilters: React.FC<Props> = ({ children, hasFiltersApplied }: Props) => {
  const classes = [ css.base ];

  if (hasFiltersApplied) classes.push(css.filtersApplied);

  return (
    <div className={classes.join(' ')}>
      <Dropdown
        content={<div className={css.modal}>{children}</div>}
        disableAutoDismiss
        offset={{ x: 0, y: 8 }}
        placement={Placement.BottomRight}>
        <Button className={css.filtersButton}>Filters</Button>
      </Dropdown>
    </div>
  );
};

export default ResponsiveFilters;
