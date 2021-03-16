import { Tooltip } from 'antd';
import React, { PropsWithChildren, useCallback, useState } from 'react';

import { isString } from 'utils/data';
import { toHtmlId } from 'utils/string';

import Icon from './Icon';
import css from './Section.module.scss';

interface Props {
  bodyBorder?: boolean;
  divider?: boolean;
  filters?: React.ReactNode;
  hideTitle?: boolean;
  id?: string;
  maxHeight?: boolean;
  options?: React.ReactNode;
  title: string | React.ReactElement;
}

const defaultProps = { divider: false };

const Section: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  const id = props.id || (isString(props.title) ? toHtmlId(props.title as string) : undefined);
  const classes = [ css.base ];
  const [ showFilters, setShowFilters ] = useState(true);

  if (props.bodyBorder) classes.push(css.bodyBorder);
  if (props.divider) classes.push(css.divider);
  if (props.filters) classes.push(css.filters);
  if (props.maxHeight) classes.push(css.maxHeight);
  if (showFilters) classes.push(css.showFilters);

  const handleFilterToggle = useCallback(() => setShowFilters(prev => !prev), []);

  return (
    <section className={classes.join(' ')} id={id}>
      <div className={css.header}>
        {!props.hideTitle && <h5 className={css.title}>{props.title}</h5>}
        {props.options && <div className={css.options}>{props.options}</div>}
        {props.filters && (
          <Tooltip placement="top" title="Toggle Filter">
            <button
              aria-label="Toggle Filter"
              className={css.filterToggle}
              onClick={handleFilterToggle}>
              <Icon name={showFilters ? 'close' : 'filter'} />
            </button>
          </Tooltip>
        )}
      </div>
      {props.filters && (
        <div className={css.filterBar}>
          {props.filters}
        </div>
      )}
      <div className={css.body}>{props.children}</div>
    </section>
  );
};

Section.defaultProps = defaultProps;

export default Section;
