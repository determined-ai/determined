import React, { useEffect, useRef, useState } from 'react';

import Dropdown, { Placement } from 'components/Dropdown';
import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import useResize from 'hooks/useResize';

import css from './ResponsiveFilters.module.scss';

interface Props {
  children: React.ReactNode;
  hasFiltersApplied?: boolean;
}

const ResponsiveFilters: React.FC<Props> = ({ children, hasFiltersApplied }: Props) => {
  const container = useRef<HTMLDivElement>(null);
  const resize = useResize(container);
  const [isCollapsed, setIsCollapsed] = useState(false);
  const [initVisible, setInitVisible] = useState(true);
  const classes = [css.base];

  if (hasFiltersApplied) classes.push(css.filtersApplied);
  if (isCollapsed) {
    classes.push('responsive-filters-collapsed');
    classes.push(css.collapsed);
  }

  /*
   * If the height of the container is more than 32,
   * it means that the filter options are wrapping and
   * needs to collapse into a filter/dropdown view.
   */
  useEffect(() => {
    if (!isCollapsed && resize.height > 48) {
      setInitVisible(false);
      setIsCollapsed(true);
    }
  }, [isCollapsed, resize.height]);

  const content = <div className={css.content}>{children}</div>;

  return (
    <div className={classes.join(' ')} ref={container}>
      {isCollapsed ? (
        <Dropdown
          content={content}
          disableAutoDismiss
          initVisible={initVisible}
          offset={{ x: 0, y: 8 }}
          placement={Placement.BottomRight}>
          <div className={css.filtersButtonDesktop}>
            <Button>Filters</Button>
          </div>
          <div className={css.filtersButtonMobile}>
            <Button icon={<Icon name="filter" />} />
          </div>
        </Dropdown>
      ) : (
        content
      )}
    </div>
  );
};

export default ResponsiveFilters;
