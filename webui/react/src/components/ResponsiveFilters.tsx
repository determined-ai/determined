import React, { useEffect, useRef, useState } from 'react';

import Button from 'components/kit/Button';
import Dropdown from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import css from 'components/ResponsiveFilters.module.scss';
import useResize from 'hooks/useResize';

interface Props {
  children: React.ReactNode;
  hasFiltersApplied?: boolean;
}

const ResponsiveFilters: React.FC<Props> = ({ children, hasFiltersApplied }: Props) => {
  const container = useRef<HTMLDivElement>(null);
  const resize = useResize(container);
  const [isCollapsed, setIsCollapsed] = useState(false);
  const classes = [css.base];
  const contentClasses = [css.content];

  if (hasFiltersApplied) classes.push(css.filtersApplied);
  if (isCollapsed) contentClasses.push(css.collapsed);

  /**
   * If the height of the container is more than 48,
   * it means that the filter options are wrapping and
   * needs to collapse into a filter/dropdown view.
   */
  useEffect(() => {
    if (!isCollapsed && resize.height > 48) setIsCollapsed(true);
  }, [isCollapsed, resize.height]);

  const content = <div className={contentClasses.join(' ')}>{children}</div>;

  return (
    <div className={classes.join(' ')} ref={container}>
      {isCollapsed ? (
        <Dropdown content={content}>
          <div>
            <div className={css.filtersButtonDesktop}>
              <Button>Filters</Button>
            </div>
            <div className={css.filtersButtonMobile}>
              <Button icon={<Icon name="filter" title="Filters" />} />
            </div>
          </div>
        </Dropdown>
      ) : (
        content
      )}
    </div>
  );
};

export default ResponsiveFilters;
