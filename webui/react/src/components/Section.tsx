import classnames from 'classnames';
import React, { PropsWithChildren } from 'react';

import Spinner from 'shared/components/Spinner/Spinner';
import { isString } from 'shared/utils/data';
import { generateAlphaNumeric, toHtmlId } from 'shared/utils/string';

import css from './Section.module.scss';

interface Props {
  bodyBorder?: boolean;
  bodyDynamic?: boolean;
  bodyNoPadding?: boolean;
  bodyScroll?: boolean;
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

const Section: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  const defaultId = isString(props.title) ? toHtmlId(props.title) : generateAlphaNumeric();
  const id = props.id || defaultId;
  const classes = classnames(
    css.base,
    {
      [css.bodyBorder]: props.bodyBorder,
      [css.bodyDynamic]: props.bodyDynamic,
      [css.bodyNoPadding]: props.bodyNoPadding,
      [css.bodyScroll]: props.bodyScroll,
      [css.divider]: props.divider,
      [css.filters]: props.filters,
      [css.maxHeight]: props.maxHeight,
    },
  );
  const titleClasses = classnames(css.title, { [css.string]: typeof props.title === 'string' });

  return (
    <section className={classes} id={id}>
      {(props.title || props.options) && (
        <div className={css.header}>
          {props.title && !props.hideTitle &&
            <h5 className={titleClasses}>{props.title}</h5>}
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
