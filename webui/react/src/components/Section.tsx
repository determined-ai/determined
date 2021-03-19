import React, { PropsWithChildren, useCallback, useState } from 'react';

import { isString } from 'utils/data';
import { generateAlphaNumeric, toHtmlId } from 'utils/string';

import css from './Section.module.scss';

interface Props {
  bodyBorder?: boolean;
  bodyScroll?: boolean;
  divider?: boolean;
  filters?: React.ReactNode;
  hideTitle?: boolean;
  id?: string;
  maxHeight?: boolean;
  noBodyPadding?: boolean;
  options?: React.ReactNode;
  title?: string | React.ReactElement;
}

const defaultProps = { divider: false };

const Section: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  const defaultId = isString(props.title) ? toHtmlId(props.title) : generateAlphaNumeric();
  const id = props.id || defaultId;
  const classes = [ css.base ];

  if (props.bodyBorder) classes.push(css.bodyBorder);
  if (props.bodyScroll) classes.push(css.bodyScroll);
  if (props.divider) classes.push(css.divider);
  if (props.filters) classes.push(css.filters);
  if (props.maxHeight) classes.push(css.maxHeight);
  if (props.noBodyPadding) classes.push(css.noBodyPadding);

  return (
    <section className={classes.join(' ')} id={id}>
      {(props.title || props.options) && (
        <div className={css.header}>
          {props.title && <h5 className={css.title}>{props.title}</h5>}
          {props.options && <div className={css.options}>{props.options}</div>}
        </div>
      )}
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
