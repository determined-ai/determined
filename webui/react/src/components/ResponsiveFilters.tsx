import { Button } from 'antd';
import React from 'react';

import Dropdown, { Placement } from 'components/Dropdown';
import useResize from 'hooks/useResize';

import css from './ResponsiveFilters.module.scss';

interface Props {
  children: React.ReactNode;
  hasFiltersApplied?: boolean;
}

const BREAKPOINT = 768;

const ResponsiveFilters: React.FC<Props> = ({ children, hasFiltersApplied }: Props) => {
  const resize = useResize();
  const classes = [ css.base ];

  if (hasFiltersApplied) classes.push(css.filtersApplied);

  const content = <div className={css.content}>{children}</div>;
  const wrappedContent = resize.width < BREAKPOINT ? (
    <Dropdown
      content={content}
      disableAutoDismiss
      offset={{ x: 0, y: 8 }}
      placement={Placement.BottomRight}>
      <Button className={css.filtersButton}>Filters</Button>
    </Dropdown>
  ) : content;

  return <div className={classes.join(' ')}>{wrappedContent}</div>;
};

export default ResponsiveFilters;
