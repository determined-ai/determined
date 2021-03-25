import React, { PropsWithChildren } from 'react';

import { isString } from 'utils/data';
import { generateAlphaNumeric, toHtmlId } from 'utils/string';

import css from './Section.module.scss';
import Spinner from './Spinner';

interface Props {
  bodyBorder?: boolean;
  bodyNoPadding?: boolean;
  bodyScroll?: boolean;
  divider?: boolean;
  filters?: React.ReactNode;
  hideTitle?: boolean;
  id?: string;
  loading?: boolean;
  maxHeight?: boolean;
  options?: React.ReactNode;
  title?: string | React.ReactElement;
}

const defaultProps = { divider: false };

const Section: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  const defaultId = isString(props.title) ? toHtmlId(props.title) : generateAlphaNumeric();
  const id = props.id || defaultId;
  const classes = [ css.base ];

  if (props.bodyBorder) classes.push(css.bodyBorder);
  if (props.bodyNoPadding) classes.push(css.bodyNoPadding);
  if (props.bodyScroll) classes.push(css.bodyScroll);
  if (props.divider) classes.push(css.divider);
  if (props.filters) classes.push(css.filters);
  if (props.maxHeight) classes.push(css.maxHeight);

  return (
    <section className={classes.join(' ')} id={id}>
      {(props.title || props.options) && (
        <div className={css.header}>
          {props.title && <h5 className={css.title}>{props.title}</h5>}
          {props.options && <div className={css.options}>{props.options}</div>}
        </div>
      )}
      {!props.loading && props.filters && (
        <div className={css.filterBar}>
          {props.filters}
        </div>
      )}
      <div className={css.body}>
        {!props.loading && props.children}
        {props.loading && <Spinner />}
      </div>
    </section>
  );
};

Section.defaultProps = defaultProps;

export default Section;
