import React from 'react';

import Spinner from 'shared/components/Spinner/Spinner';
import { isString } from 'shared/utils/data';
import { generateAlphaNumeric, toHtmlId } from 'shared/utils/string';

import css from './Section.module.scss';

interface Props {
  bodyBorder?: boolean;
  bodyDynamic?: boolean;
  bodyNoPadding?: boolean;
  bodyScroll?: boolean;
  children: React.ReactNode;
  className?: string;
  divider?: boolean;
  filters?: React.ReactNode;
  hideTitle?: boolean;
  id?: string;
  loading?: boolean;
  maxHeight?: boolean;
  options?: React.ReactNode;
  title?: string | React.ReactNode;
}

const defaultProps = { divider: false };

const Section: React.FC<Props> = ({ className = '', ...props }: Props) => {
  const defaultId = isString(props.title) ? toHtmlId(props.title) : generateAlphaNumeric();
  const id = props.id || defaultId;
  const classes = [ css.base, className ];
  const titleClasses = [ css.title ];

  if (props.bodyBorder) classes.push(css.bodyBorder);
  if (props.bodyDynamic) classes.push(css.bodyDynamic);
  if (props.bodyNoPadding) classes.push(css.bodyNoPadding);
  if (props.bodyScroll) classes.push(css.bodyScroll);
  if (props.divider) classes.push(css.divider);
  if (props.filters) classes.push(css.filters);
  if (props.maxHeight) classes.push(css.maxHeight);
  if (typeof props.title === 'string') titleClasses.push(css.string);

  return (
    <section className={classes.join(' ')} id={id}>
      {(props.title || props.options) && (
        <div className={css.header}>
          {props.title && !props.hideTitle &&
            <h5 className={titleClasses.join(' ')}>{props.title}</h5>}
          {props.options && <div className={css.options}>{props.options}</div>}
        </div>
      )}
      {props.filters && (
        <div className={css.filterBar}>
          {props.filters}
        </div>
      )}
      <div className={css.body}>
        <Spinner spinning={!!props.loading}>{props.children}</Spinner>
      </div>
    </section>
  );
};

Section.defaultProps = defaultProps;

export default Section;
