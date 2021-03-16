import React, { PropsWithChildren, useCallback, useState } from 'react';

import useStorage from 'hooks/useStorage';
import { isString } from 'utils/data';
import { generateAlphaNumeric, toHtmlId } from 'utils/string';

import IconButton from './IconButton';
import css from './Section.module.scss';

interface Props {
  bodyBorder?: boolean;
  divider?: boolean;
  filters?: React.ReactNode;
  hideTitle?: boolean;
  id?: string;
  maxHeight?: boolean;
  noBodyPadding?: boolean;
  options?: React.ReactNode;
  title: string | React.ReactElement;
}

const defaultProps = { divider: false };
const STORAGE_PATH = 'section';

const Section: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  const defaultId = isString(props.title) ? toHtmlId(props.title) : generateAlphaNumeric();
  const id = props.id || defaultId;
  const storage = useStorage(STORAGE_PATH);
  const defaultShowFilters = storage.getWithDefault(id, true);
  const [ showFilters, setShowFilters ] = useState(defaultShowFilters);
  const classes = [ css.base ];

  if (props.bodyBorder) classes.push(css.bodyBorder);
  if (props.divider) classes.push(css.divider);
  if (props.filters) classes.push(css.filters);
  if (props.maxHeight) classes.push(css.maxHeight);
  if (props.noBodyPadding) classes.push(css.noBodyPadding);
  if (showFilters) classes.push(css.showFilters);

  const handleFilterToggle = useCallback(() => {
    setShowFilters(prev => {
      storage.set(id, !prev);
      return !prev;
    });
  }, [ id, storage ]);

  return (
    <section className={classes.join(' ')} id={id}>
      <div className={css.header}>
        {!props.hideTitle && <h5 className={css.title}>{props.title}</h5>}
        {props.options && <div className={css.options}>{props.options}</div>}
        {props.filters && <IconButton
          className={css.filterToggle}
          icon={showFilters ? 'close' : 'filter'}
          label={showFilters ? 'Close Filter Bar' : 'Open Filter Bar'}
          onClick={handleFilterToggle} />}
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
